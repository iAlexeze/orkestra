//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

// ── idp ─────────────────────────────────────────────────────────────────────

var idpCmd = &cobra.Command{
	Use:   "idp",
	Short: "Inspect and validate IDP configurations",
	Long: `Inspect and validate Internal Developer Platform (IDP) configurations.

The IDP is the contract between platform teams and developers. It defines
what fields developers can submit, what the gateway builds, and what
callers see.

Subcommands:
  validate    Validate IDP configuration in a Katalog
  schema      Show the flat schema for an IDP target
  fields      List all IDP fields with their paths and types
  tokens      Show token permissions for a CRD
  targets     List all IDP targets in a Katalog
  can-i       Check if a token can perform an operation
  response    Show the IDP response configuration`,
}

// ── idp validate ────────────────────────────────────────────────────────────

var idpValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate IDP configuration",
	Long: `Validate IDP configuration in a Katalog.

This runs the same IDP-specific validations as ork validate, but only for
IDP concerns: fields, paths, tokens, response config, and namespace rules.

It does not check the full Katalog schema — only the IDP portions.

With --full, shows a detailed breakdown of the IDP configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		full, _ := cmd.Flags().GetBool("full")

		if err := k.ValidateIDP(); err != nil {
			return err
		}

		if full {
			printIDPValidationSummary(k)
		} else {
			fmt.Printf("\n%s IDP configuration is valid\n", successMark())
		}

		return nil
	},
}

// ── idp schema ─────────────────────────────────────────────────────────────

var idpSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Show the flat schema for an IDP target",
	Long: `Show the flat schema for an IDP target.

This displays the fields that callers can submit for a target, including
their labels, types, enums, and paths.

The output is the same flat field structure returned by the gateway's
GET /api/v1/schema?target=<t> endpoint.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		kind, _ := cmd.Flags().GetString("kind")
		name, _ := cmd.Flags().GetString("name")

		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		crd, err := resolveCRD(k, target, kind, name)
		if err != nil {
			return err
		}

		if !crd.IDPEnabled() {
			return fmt.Errorf("CRD %q is not IDP-enabled", crd.Name)
		}

		fields := crd.IDPFields()
		if len(fields) == 0 {
			fmt.Printf("\nNo fields defined for target %q\n", target)
			return nil
		}

		fmt.Printf("\nSchema for: %s (target: %s)\n", crd.Name, crd.IDPTarget())
		fmt.Printf("%s\n", strings.Repeat("─", 70))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "FIELD\tLABEL\tTYPE\tPATH\tREQUIRED")

		for name, config := range fields {
			specPath := config.SpecPath(name)
			required := ""
			if config.Required {
				required = "✓"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name,
				config.Label,
				config.FieldType(),
				specPath,
				required,
			)
		}
		w.Flush()
		fmt.Println()
		return nil
	},
}

// ── idp fields ─────────────────────────────────────────────────────────────

var idpFieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "List IDP fields with their paths and types",
	Long: `List IDP fields in a Katalog with their paths and types.

This shows fields declared in idp.fields and idp.additionalFields
across all IDP-enabled CRDs.

With --target, --kind, or --name, shows fields for a specific CRD.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		kind, _ := cmd.Flags().GetString("kind")
		name, _ := cmd.Flags().GetString("name")

		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		// ── If a specific CRD is requested ──────────────────────────────────
		if target != "" || kind != "" || name != "" {
			crd, err := resolveCRD(k, target, kind, name)
			if err != nil {
				return err
			}

			if !crd.IDPEnabled() {
				return fmt.Errorf("CRD %q is not IDP-enabled", crd.Name)
			}

			fields := crd.IDPFields()
			if len(fields) == 0 {
				fmt.Printf("\nNo fields defined for CRD %q\n", crd.Name)
				return nil
			}

			fmt.Printf("\nFields for: %s (target: %s)\n", crd.Name, crd.IDPTarget())
			fmt.Printf("%s\n", strings.Repeat("─", 70))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "FIELD\tTYPE\tPATH\tSOURCE\tREQUIRED")

			// ── Sort fields by name ─────────────────────────────────────────────────────
			type fieldEntry struct {
				Name     string
				Config   orktypes.IDPFieldConfig
				SpecPath string
				Source   string
			}

			entries := make([]fieldEntry, 0, len(fields))
			for name, config := range fields {
				specPath := config.SpecPath(name)
				source := "spec"
				if _, ok := crd.AdditionalLabelFields()[name]; ok {
					source = "label"
				} else if _, ok := crd.AdditionalAnnotationFields()[name]; ok {
					source = "annotation"
				}
				entries = append(entries, fieldEntry{
					Name:     name,
					Config:   config,
					SpecPath: specPath,
					Source:   source,
				})
			}

			// Sort by name only
			// Debug: print entries before sorting
			for _, e := range entries {
				fmt.Printf("BEFORE: %s\n", e.Name)
			}

			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Name < entries[j].Name
			})

			// Debug: print entries after sorting
			for _, e := range entries {
				fmt.Printf("AFTER: %s\n", e.Name)
			}

			// ─── Print sorted entries ──────────────────────────────────────────────────
			for _, entry := range entries {
				required := ""
				if entry.Config.Required {
					required = "✓"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					entry.Name,
					entry.Config.FieldType(),
					entry.SpecPath,
					entry.Source,
					required,
				)
			}
			w.Flush()
			fmt.Println()
			return nil
		}

		// ── All fields across all CRDs ──────────────────────────────────────
		var totalFields int
		fmt.Printf("\nIDP Fields\n")
		fmt.Printf("%s\n", strings.Repeat("─", 70))

		for _, crd := range k.IDPEnabledCRDs() {
			if crd == nil {
				continue
			}
			fields := crd.IDPFields()
			if len(fields) == 0 {
				continue
			}

			fmt.Printf("\nCRD: %s (target: %s)\n", crd.Name, crd.IDPTarget())
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  FIELD\tTYPE\tPATH\tSOURCE")

			for name, config := range fields {
				specPath := config.SpecPath(name)
				source := "spec"
				if _, ok := crd.AdditionalLabelFields()[name]; ok {
					source = "label"
				} else if _, ok := crd.AdditionalAnnotationFields()[name]; ok {
					source = "annotation"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
					name,
					config.FieldType(),
					specPath,
					source,
				)
				totalFields++
			}
			w.Flush()
		}

		if totalFields == 0 {
			fmt.Println("\nNo IDP fields found.")
		} else {
			fmt.Printf("\nTotal: %d fields across %d CRDs\n", totalFields, len(k.IDPEnabledCRDs()))
		}
		fmt.Println()
		return nil
	},
}

// ── idp tokens ─────────────────────────────────────────────────────────────

var idpTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Show token permissions for a CRD",
	Long: `Show token permissions for an IDP-enabled CRD.

This displays the allowedTokens configuration, including which tokens
have access, their permissions (global/schema/resources), and namespace
restrictions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		kind, _ := cmd.Flags().GetString("kind")
		name, _ := cmd.Flags().GetString("name")

		if target == "" && kind == "" && name == "" {
			return fmt.Errorf("one of --target, --kind, or --name is required")
		}

		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		crd, err := resolveCRD(k, target, kind, name)
		if err != nil {
			return err
		}

		if !crd.IDPEnabled() {
			return fmt.Errorf("CRD %q is not IDP-enabled", crd.Name)
		}

		if !crd.HasIDPTokenRestrictions() {
			fmt.Printf("\nNo token restrictions for CRD %q\n", crd.Name)
			return nil
		}

		fmt.Printf("\nToken permissions for CRD: %s (target: %s)\n", crd.Name, crd.IDPTarget())
		fmt.Printf("%s\n", strings.Repeat("─", 70))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TOKEN\tGLOBAL\tSCHEMA\tRESOURCES\tNAMESPACES")

		for tokenName, perms := range crd.IDP.AllowedTokensMap() {
			global := strings.Join(perms.Permissions.Global, ",")
			schema := strings.Join(perms.Permissions.Schema, ",")
			resources := strings.Join(perms.Permissions.Resources, ",")
			namespaces := strings.Join(perms.Namespaces, ",")
			if namespaces == "" {
				namespaces = "*"
			}
			if global == "" && schema == "" && resources == "" {
				global = "—"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				tokenName,
				global,
				schema,
				resources,
				namespaces,
			)
		}
		w.Flush()
		fmt.Println()
		return nil
	},
}

// ── idp targets ─────────────────────────────────────────────────────────────

var idpTargetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List all IDP targets in a Katalog",
	Long: `List all IDP-enabled targets in a Katalog.

This shows each target, its CRD kind, and whether it has fields defined.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		var entries []struct {
			Target   string
			Kind     string
			Fields   int
			HasToken bool
		}

		for _, crd := range k.IDPEnabledCRDs() {
			if crd == nil {
				continue
			}
			entries = append(entries, struct {
				Target   string
				Kind     string
				Fields   int
				HasToken bool
			}{
				Target:   crd.IDPTarget(),
				Kind:     crd.Kind(),
				Fields:   len(crd.IDPFields()),
				HasToken: crd.HasIDPTokenRestrictions(),
			})
		}

		if len(entries) == 0 {
			fmt.Println("\nNo IDP targets found.")
			return nil
		}

		fmt.Printf("\nIDP Targets\n")
		fmt.Printf("%s\n", strings.Repeat("─", 50))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TARGET\tKIND\tFIELDS\tTOKENS")

		for _, e := range entries {
			tokens := "no"
			if e.HasToken {
				tokens = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
				e.Target,
				e.Kind,
				e.Fields,
				tokens,
			)
		}
		w.Flush()
		fmt.Printf("\n%d target(s)\n", len(entries))
		fmt.Println()
		return nil
	},
}

// ── idp can-i ──────────────────────────────────────────────────────────────

var idpCanICmd = &cobra.Command{
	Use:   "can-i",
	Short: "Check if a token can perform an operation",
	Long: `Check if a token can perform an operation on a target.

This evaluates the same permission checks that the gateway applies to
incoming Apply API requests. It considers:

  - Token existence in gateway.applyAPI.auth.tokens
  - Token permissions (global/schema/resources scopes)
  - Namespace restrictions
  - Target existence

Useful for debugging permission issues and auditing token capabilities.

Examples:
  ork idp can-i --token control-center --target smartapp --operation create
  ork idp can-i --token ci-pipeline --target smartapp --operation delete --namespace staging
  ork idp can-i --token monitoring --target smartapp --operation list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Flags().GetString("token")
		op, _ := cmd.Flags().GetString("operation")
		namespace, _ := cmd.Flags().GetString("namespace")
		classFlag, _ := cmd.Flags().GetString("class")
		target, _ := cmd.Flags().GetString("target")
		kind, _ := cmd.Flags().GetString("kind")
		name, _ := cmd.Flags().GetString("name")

		if token == "" {
			return fmt.Errorf("--token is required")
		}
		if op == "" {
			return fmt.Errorf("--operation is required (%s)", validIDPOperations)
		}
		if !orktypes.IsValidIDPOperation(op) {
			return fmt.Errorf("%s --operation must be one of %s", failureMark(), validIDPOperations)
		}
		if classFlag != "" {
			if !orktypes.IsValidIDPEndpointClass(classFlag) {
				return fmt.Errorf("%s --class must be one of %s", failureMark(), validIDPEndpointClasses)
			}
		}

		if target == "" && kind == "" && name == "" {
			return fmt.Errorf("one of --target, --kind, or --name is required")
		}

		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		crd, err := resolveCRD(k, target, kind, name)
		if err != nil {
			return err
		}

		if !crd.IDPEnabled() {
			return fmt.Errorf("CRD %q is not IDP-enabled", crd.Name)
		}

		// Check if the token exists in the gateway config
		gatewayTokens := k.GatewayTokenNames()
		tokenExists := false
		for _, t := range gatewayTokens {
			if t == token {
				tokenExists = true
				break
			}
		}
		if !tokenExists {
			printCanIResult(false, token, op, crd, namespace,
				fmt.Sprintf("token %q is not defined in gateway.applyAPI.auth.tokens", token),
				gatewayTokens)
			return nil
		}

		// Check if the token has any restrictions
		if !crd.HasIDPTokenRestrictions() {
			printCanIResult(true, token, op, crd, namespace,
				"no token restrictions declared — all tokens are allowed",
				nil)
			return nil
		}

		// Get the token's permissions
		perms, ok := crd.IDP.AllowedTokensMap()[token]
		if !ok {
			printCanIResult(false, token, op, crd, namespace,
				fmt.Sprintf("token %q is not listed in allowedTokens for CRD %q", token, crd.Name),
				nil)
			return nil
		}

		// Determine endpoint class
		class := orktypes.IDPClassResources
		if classFlag == strings.ToLower("schema") {
			class = orktypes.IDPClassSchema
		}

		// Check namespace restrictions
		// ── 1. Check namespace against CRD allowed namespaces ──────────────────────
		if namespace != "" && crd.IsNamespaceRestricted() {
			if !crd.IsNamespaceAuthorized(namespace) {
				printCanIResult(false, token, op, crd, namespace,
					fmt.Sprintf("namespace %q is not allowed for CRD %q", namespace, crd.Name),
					crd.AllowedNamespaces)
				return nil
			}
		}

		// ── 2. Check namespace against token restrictions ──────────────────────────
		if namespace != "" && perms.IsNamespaceRestricted() {
			if !perms.HasNamespace(namespace) {
				printCanIResult(false, token, op, crd, namespace,
					fmt.Sprintf("token %q is not allowed in namespace %q", token, namespace),
					perms.Namespaces)
				return nil
			}
		}

		// Check operation permission
		if perms.HasOperation(class, op) {
			printCanIResult(true, token, op, crd, namespace,
				fmt.Sprintf("token %q has %q permission", token, op),
				nil)
			return nil
		}

		// Token has permissions but not this specific one
		var has []string
		if perms.HasGlobalPermissions() {
			has = append(has, "global: "+strings.Join(perms.Permissions.Global, ","))
		}
		if perms.HasSchemaPermissions() {
			has = append(has, "schema: "+strings.Join(perms.Permissions.Schema, ","))
		}
		if perms.HasResourcesPermissions() {
			has = append(has, "resources: "+strings.Join(perms.Permissions.Resources, ","))
		}

		printCanIResult(false, token, op, crd, namespace,
			fmt.Sprintf("token %q does not have %q permission for %s class", token, op, class),
			has)
		return nil
	},
}

// ── idp response ────────────────────────────────────────────────────────────

var idpResponseCmd = &cobra.Command{
	Use:   "response",
	Short: "Show the IDP response configuration",
	Long: `Show the IDP response configuration for a target.

This displays what callers will see in the Apply API response based on
idp.config.response. It shows:

  - default: true/false
  - Payload fields (with their template expressions)
  - Excluded paths
  - Poll URL configuration

No cluster access is required — this reads the Katalog directly.

Examples:
  ork idp response --target smartapp
  ork idp response --target app --preview`,
	RunE: func(cmd *cobra.Command, args []string) error {
		preview, _ := cmd.Flags().GetBool("preview")
		target, _ := cmd.Flags().GetString("target")
		kind, _ := cmd.Flags().GetString("kind")
		name, _ := cmd.Flags().GetString("name")

		if target == "" && kind == "" && name == "" {
			return fmt.Errorf("one of --target, --kind, or --name is required")
		}

		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		crd, err := resolveCRD(k, target, kind, name)
		if err != nil {
			return err
		}

		if !crd.IDPEnabled() {
			return fmt.Errorf("CRD %q is not IDP-enabled", crd.Name)
		}

		cfg := crd.GetIDPResponseConfig()
		if cfg == nil {
			fmt.Printf("\nNo response configuration for CRD %q\n", crd.Name)
			fmt.Println("  Add idp.config.response to customize what callers see.")
			fmt.Println()
			return nil
		}

		fmt.Printf("\nResponse configuration for: %s (target: %s)\n", crd.Name, crd.IDPTarget())
		fmt.Printf("%s\n", strings.Repeat("─", 70))

		fmt.Printf("\ndefault: %v\n", cfg.UseDefault())

		if cfg.HasPayload() {
			fmt.Println("\nPayload fields:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  FIELD\tEXPRESSION")
			for key, expr := range cfg.Payload {
				display := expr
				if len(display) > 55 {
					display = display[:52] + "..."
				}
				fmt.Fprintf(w, "  %s\t%s\n", key, display)
			}
			w.Flush()
		} else {
			fmt.Println("\nPayload fields: none")
		}

		if cfg.HasExclude() {
			fmt.Println("\nExcluded paths:")
			for _, path := range cfg.Exclude {
				fmt.Printf("  ✗ %s\n", path)
			}
		} else {
			fmt.Println("\nExcluded paths: none")
		}

		if cfg.HasPoll() {
			fmt.Println("\nPoll URL:")
			if cfg.Poll.URL != "" {
				fmt.Printf("  url:   %q\n", cfg.Poll.URL)
			}
			if cfg.Poll.Field != "" {
				fmt.Printf("  field: %q\n", cfg.Poll.Field)
			}
		}

		if preview {
			fmt.Printf("\n%s\n", strings.Repeat("─", 70))
			fmt.Println("\nResponse preview (templates shown as-is, not resolved):")

			if cfg.HasPayload() {
				fmt.Println()
				fmt.Printf("%s\n", blue("{"))

				keys := make([]string, 0, len(cfg.Payload))
				for k := range cfg.Payload {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				for i, key := range keys {
					expr := cfg.Payload[key]
					comma := ","
					if i == len(keys)-1 {
						comma = ""
					}
					fmt.Printf("  %s: %s%s\n",
						blue(fmt.Sprintf("%q", key)),
						green(fmt.Sprintf("%q", expr)),
						white(comma),
					)
				}
				fmt.Printf("%s\n", blue("}"))
			} else {
				fmt.Println("\n  No payload fields configured.")
			}

			if cfg.HasExclude() {
				fmt.Println("\nExcluded fields:")
				for _, path := range cfg.Exclude {
					fmt.Printf("  ✗ %s\n", path)
				}
			}
		}

		fmt.Println()
		return nil
	},
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// resolveCRD resolves a CRD by target, kind, or name.
func resolveCRD(kat *katalog.Katalog, target, kind, name string) (*orktypes.CRDEntry, error) {
	var crd *orktypes.CRDEntry

	switch {
	case target != "":
		crd = kat.LookupByTarget(target)
		if crd == nil {
			return nil, fmt.Errorf("%s target %q not found", failureMark(), target)
		}
	case kind != "":
		crd = kat.LookupByKind(kind)
		if crd == nil {
			return nil, fmt.Errorf("%s kind %q not found", failureMark(), kind)
		}
	case name != "":
		entry, ok := kat.CRDEntry(name)
		if !ok {
			return nil, fmt.Errorf("%s CRD %q not found", failureMark(), name)
		}
		crd = &entry
	default:
		return nil, fmt.Errorf("one of --target, --kind, or --name is required")
	}

	return crd, nil
}

// printCanIResult prints the permission check result.
func printCanIResult(allowed bool, token, op string, crd *orktypes.CRDEntry, namespace, reason string, details []string) {
	if op == "*" {
		op = "perform all operations"
	}

	fmt.Println()
	if allowed {
		fmt.Printf("%s %s can %s on %q", successMark(), token, op, crd.IDPTarget())
		if namespace != "" {
			fmt.Printf(" in namespace %q", namespace)
		}
		fmt.Println()
	} else {
		fmt.Printf("%s %s cannot %s on %q", failureMark(), token, op, crd.IDPTarget())
		if namespace != "" {
			fmt.Printf(" in namespace %q", namespace)
		}
		fmt.Println()
		fmt.Printf("  Reason: %s\n", reason)
		if len(details) > 0 {
			fmt.Printf("  Available:\n")
			for _, detail := range details {
				fmt.Printf("    - %s\n", detail)
			}
		}
	}
	fmt.Println()
}

// ── init ────────────────────────────────────────────────────────────────────

func init() {
	// Validate
	idpValidateCmd.Flags().Bool("full", false, "Show a detailed breakdown of the IDP configuration")
	idpCmd.AddCommand(idpValidateCmd)

	// Schema
	idpSchemaCmd.Flags().StringP("target", "t", "", "Target to show schema for")
	idpSchemaCmd.Flags().StringP("kind", "k", "", "Kind to show schema for")
	idpSchemaCmd.Flags().StringP("name", "n", "", "CRD name to show schema for")
	idpCmd.AddCommand(idpSchemaCmd)

	// Fields
	idpFieldsCmd.Flags().StringP("target", "t", "", "Target to show fields for")
	idpFieldsCmd.Flags().StringP("kind", "k", "", "Kind to show fields for")
	idpFieldsCmd.Flags().StringP("name", "n", "", "CRD name to show fields for")
	idpCmd.AddCommand(idpFieldsCmd)

	// Tokens
	idpTokensCmd.Flags().StringP("target", "t", "", "Target to show tokens for")
	idpTokensCmd.Flags().StringP("kind", "k", "", "Kind to show tokens for")
	idpTokensCmd.Flags().StringP("name", "n", "", "CRD name to show tokens for")
	idpCmd.AddCommand(idpTokensCmd)

	// Targets
	idpCmd.AddCommand(idpTargetsCmd)

	// CanI
	idpCanICmd.Flags().StringP("token", "T", "", "Token name to check")
	idpCanICmd.Flags().StringP("target", "t", "", "Target to check")
	idpCanICmd.Flags().StringP("kind", "k", "", "Kind to check")
	idpCanICmd.Flags().StringP("name", "n", "", "CRD name to check")
	idpCanICmd.Flags().StringP("operation", "o", "", "Operation to check ("+validIDPOperations+")")
	idpCanICmd.Flags().StringP("namespace", "N", "", "Namespace to check (default: all namespaces)")
	idpCanICmd.Flags().StringP("class", "c", "resources", "Endpoint class to check (resources, schema)")
	idpCmd.AddCommand(idpCanICmd)

	// Response
	idpResponseCmd.Flags().StringP("target", "t", "", "Target to show response for")
	idpResponseCmd.Flags().StringP("kind", "k", "", "Kind to show response for")
	idpResponseCmd.Flags().StringP("name", "n", "", "CRD name to show response for")
	idpResponseCmd.Flags().BoolP("preview", "p", false, "Show a sample response preview")
	idpCmd.AddCommand(idpResponseCmd)

	rootCmd.AddCommand(idpCmd)

	// Shadow global flags
	idpCmds := []*cobra.Command{
		idpCmd,
		idpValidateCmd,
		idpSchemaCmd,
		idpFieldsCmd,
		idpTokensCmd,
		idpTargetsCmd,
		idpCanICmd,
		idpResponseCmd,
	}
	for _, cmd := range idpCmds {
		shadowGlobalCommandFlags(cmd)
	}
}
