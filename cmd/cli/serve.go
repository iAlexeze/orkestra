//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

// ── serve ─────────────────────────────────────────────────────────────────────

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Inspect and validate Serve configurations",
	Long: `Inspect and validate Internal Developer Platform (Serve) configurations.

The Serve is the contract between platform teams and developers. It defines
what fields developers can submit, what the gateway builds, and what
callers see.

Subcommands:
  validate    Validate Serve configuration in a Katalog
  schema      Show the flat schema for a Serve target
  fields      List all Serve fields with their paths and types
  tokens      Show token permissions for a CRD
  targets     List all Serve targets in a Katalog
  can-i       Check if a token can perform an operation
  response    Show the Serve response configuration`,
}

// ── serve validate ────────────────────────────────────────────────────────────

var serveValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Serve configuration",
	Long: `Validate Serve configuration in a Katalog.

This runs the same Serve-specific validations as ork validate, but only for
Serve concerns: fields, paths, tokens, response config, and namespace rules.

It does not check the full Katalog schema — only the Serve portions.

With --full, shows a detailed breakdown of the Serve configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		k, err := buildKatalog(cmd)
		if err != nil {
			return err
		}

		full, _ := cmd.Flags().GetBool("full")

		if err := k.ValidateServe(); err != nil {
			return err
		}

		if full {
			printServeValidationSummary(k)
		} else {
			fmt.Printf("\n%s Serve configuration is valid\n", successMark())
		}

		return nil
	},
}

// ── serve schema ─────────────────────────────────────────────────────────────

var serveSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Show the flat schema for a Serve target",
	Long: `Show the flat schema for a Serve target.

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

		if !crd.ServeEnabled() {
			return fmt.Errorf("CRD %q is not Serve-enabled", crd.Name)
		}

		fields := crd.AllServeFields()
		if len(fields) == 0 {
			fmt.Printf("\nNo fields defined for target %q\n", target)
			return nil
		}

		fmt.Printf("\nSchema for: %s (target: %s)\n", crd.Name, crd.ServeTarget())
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

// ── serve fields ─────────────────────────────────────────────────────────────

var serveFieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "List Serve fields with their paths and types",
	Long: `List Serve fields in a Katalog with their paths and types.

This shows fields declared in serve.fields and serve.additionalFields
across all Serve-enabled CRDs.

With --target, --kind, or --name, shows fields for a specific CRD.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		kind, _ := cmd.Flags().GetString("kind")
		name, _ := cmd.Flags().GetString("name")
		sortBy, _ := cmd.Flags().GetString("sort-by")

		// Validate sortBy
		if sortBy != "" && sortBy != "name" && sortBy != "order" {
			return fmt.Errorf("%s --sort-by must be 'name' or 'order'", failureMark())
		}
		if sortBy == "" {
			sortBy = "name"
		}

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

			if !crd.ServeEnabled() {
				return fmt.Errorf("CRD %q is not Serve-enabled", crd.Name)
			}

			entries := sortedFieldEntries(crd, sortBy)
			if len(entries) == 0 {
				fmt.Printf("\nNo fields defined for CRD %q\n", crd.Name)
				return nil
			}

			fmt.Printf("\nFields for: %s (target: %s)\n", crd.Name, crd.ServeTarget())
			fmt.Printf("%s\n", strings.Repeat("─", 70))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "FIELD\tTYPE\tPATH\tSOURCE\tREQUIRED")

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
		crds := k.ServeEnabledCRDs()
		sort.Slice(crds, func(i, j int) bool {
			if crds[i] == nil || crds[j] == nil {
				return false
			}
			return crds[i].Name < crds[j].Name
		})

		var totalFields int
		fmt.Printf("\nServe Fields\n")
		fmt.Printf("%s\n", strings.Repeat("─", 70))

		for _, crd := range crds {
			if crd == nil {
				continue
			}
			entries := sortedFieldEntries(crd, sortBy)
			if len(entries) == 0 {
				continue
			}

			fmt.Printf("\nCRD: %s (target: %s)\n", crd.Name, crd.ServeTarget())
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  FIELD\tTYPE\tPATH\tSOURCE")

			for _, entry := range entries {
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
					entry.Name,
					entry.Config.FieldType(),
					entry.SpecPath,
					entry.Source,
				)
				totalFields++
			}
			w.Flush()
		}

		if totalFields == 0 {
			fmt.Println("\nNo Serve fields found.")
		} else {
			fmt.Printf("\nTotal: %d fields across %d CRDs\n", totalFields, len(crds))
		}
		fmt.Println()
		return nil
	},
}

// ── serve tokens ─────────────────────────────────────────────────────────────

var serveTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Show token permissions for a CRD",
	Long: `Show token permissions for a Serve-enabled CRD.

This displays the tokens configuration, including which tokens
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

		if !crd.ServeEnabled() {
			return fmt.Errorf("CRD %q is not Serve-enabled", crd.Name)
		}

		if !crd.HasServeTokenRestrictions() {
			fmt.Printf("\nNo token restrictions for CRD %q\n", crd.Name)
			return nil
		}

		fmt.Printf("\nToken permissions for CRD: %s (target: %s)\n", crd.Name, crd.ServeTarget())
		fmt.Printf("%s\n", strings.Repeat("─", 70))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TOKEN\tGLOBAL\tSCHEMA\tRESOURCES\tNAMESPACES")

		for tokenName, perms := range crd.Serve.TokensMap() {
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

// ── serve targets ─────────────────────────────────────────────────────────────

var serveTargetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List all Serve targets in a Katalog",
	Long: `List all Serve-enabled targets in a Katalog.

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

		for _, crd := range k.ServeEnabledCRDs() {
			if crd == nil {
				continue
			}
			entries = append(entries, struct {
				Target   string
				Kind     string
				Fields   int
				HasToken bool
			}{
				Target:   crd.ServeTarget(),
				Kind:     crd.Kind(),
				Fields:   len(crd.AllServeFields()),
				HasToken: crd.HasServeTokenRestrictions(),
			})
		}

		if len(entries) == 0 {
			fmt.Println("\nNo Serve targets found.")
			return nil
		}

		fmt.Printf("\nServe Targets\n")
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

// ── serve can-i ──────────────────────────────────────────────────────────────

var serveCanICmd = &cobra.Command{
	Use:   "can-i",
	Short: "Check if a token can perform an operation",
	Long: `Check if a token can perform an operation on a target.

This evaluates the same permission checks that the gateway applies to
incoming Gateway API requests. It considers:

  - Token existence in gateway.api.auth.tokens
  - Token permissions (global/schema/resources scopes)
  - Namespace restrictions
  - Target existence

Useful for debugging permission issues and auditing token capabilities.

Examples:
  ork serve can-i --token control-center --target smartapp --operation create
  ork serve can-i --token ci-pipeline --target smartapp --operation delete --namespace staging
  ork serve can-i --token monitoring --target smartapp --operation list`,
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
			return fmt.Errorf("--operation is required (%s)", validServeOperations)
		}
		if !orktypes.IsValidServeOperation(op) {
			return fmt.Errorf("%s --operation must be one of %s", failureMark(), validServeOperations)
		}
		if classFlag != "" {
			if !orktypes.IsValidServeEndpointClass(classFlag) {
				return fmt.Errorf("%s --class must be one of %s", failureMark(), validServeEndpointClasses)
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

		if !crd.ServeEnabled() {
			return fmt.Errorf("CRD %q is not Serve-enabled", crd.Name)
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
				fmt.Sprintf("token %q is not defined in gateway.api.auth.tokens", token),
				gatewayTokens)
			return nil
		}

		// Check if the token has any restrictions
		if !crd.HasServeTokenRestrictions() {
			printCanIResult(true, token, op, crd, namespace,
				"no token restrictions declared — all tokens are allowed",
				nil)
			return nil
		}

		// Get the token's permissions
		perms, ok := crd.Serve.TokensMap()[token]
		if !ok {
			printCanIResult(false, token, op, crd, namespace,
				fmt.Sprintf("token %q is not listed in tokens for CRD %q", token, crd.Name),
				nil)
			return nil
		}

		// Determine endpoint class
		class := orktypes.ServeClassResources
		if classFlag == strings.ToLower("schema") {
			class = orktypes.ServeClassSchema
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

// ── serve response ────────────────────────────────────────────────────────────

var serveResponseCmd = &cobra.Command{
	Use:   "response",
	Short: "Show the Serve response configuration",
	Long: `Show the Serve response configuration for a target.

This displays what callers will see in the Gateway API response based on
serve.config.response. It shows:

  - default: true/false
  - Payload fields (with their template expressions)
  - Excluded paths
  - Poll URL configuration

No cluster access is required — this reads the Katalog directly.

Examples:
  ork serve response --target smartapp
  ork serve response --target app --preview`,
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

		if !crd.ServeEnabled() {
			return fmt.Errorf("CRD %q is not Serve-enabled", crd.Name)
		}

		cfg := crd.GetServeResponseConfig()
		if cfg == nil {
			fmt.Printf("\nNo response configuration for CRD %q\n", crd.Name)
			fmt.Println("  Add serve.config.response to customize what callers see.")
			fmt.Println()
			return nil
		}

		fmt.Printf("\nResponse configuration for: %s (target: %s)\n", crd.Name, crd.ServeTarget())
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

// ── init ────────────────────────────────────────────────────────────────────

func init() {
	// Validate
	serveValidateCmd.Flags().Bool("full", false, "Show a detailed breakdown of the Serve configuration")
	serveCmd.AddCommand(serveValidateCmd)

	// Schema
	serveSchemaCmd.Flags().StringP("target", "t", "", "Target to show schema for")
	serveSchemaCmd.Flags().StringP("kind", "k", "", "Kind to show schema for")
	serveSchemaCmd.Flags().StringP("name", "n", "", "CRD name to show schema for")
	serveCmd.AddCommand(serveSchemaCmd)

	// Fields
	serveFieldsCmd.Flags().StringP("target", "t", "", "Target to show fields for")
	serveFieldsCmd.Flags().StringP("kind", "k", "", "Kind to show fields for")
	serveFieldsCmd.Flags().StringP("name", "n", "", "CRD name to show fields for")
	serveFieldsCmd.Flags().String("sort-by", "name", "Sort fields by 'name' (default) or 'order'")
	serveCmd.AddCommand(serveFieldsCmd)

	// Tokens
	serveTokensCmd.Flags().StringP("target", "t", "", "Target to show tokens for")
	serveTokensCmd.Flags().StringP("kind", "k", "", "Kind to show tokens for")
	serveTokensCmd.Flags().StringP("name", "n", "", "CRD name to show tokens for")
	serveCmd.AddCommand(serveTokensCmd)

	// Targets
	serveCmd.AddCommand(serveTargetsCmd)

	// CanI
	serveCanICmd.Flags().StringP("token", "T", "", "Token name to check")
	serveCanICmd.Flags().StringP("target", "t", "", "Target to check")
	serveCanICmd.Flags().StringP("kind", "k", "", "Kind to check")
	serveCanICmd.Flags().StringP("name", "n", "", "CRD name to check")
	serveCanICmd.Flags().StringP("operation", "o", "", "Operation to check ("+validServeOperations+")")
	serveCanICmd.Flags().StringP("namespace", "N", "", "Namespace to check (default: all namespaces)")
	serveCanICmd.Flags().StringP("class", "c", "resources", "Endpoint class to check (resources, schema)")
	serveCmd.AddCommand(serveCanICmd)

	// Response
	serveResponseCmd.Flags().StringP("target", "t", "", "Target to show response for")
	serveResponseCmd.Flags().StringP("kind", "k", "", "Kind to show response for")
	serveResponseCmd.Flags().StringP("name", "n", "", "CRD name to show response for")
	serveResponseCmd.Flags().BoolP("preview", "p", false, "Show a sample response preview")
	serveCmd.AddCommand(serveResponseCmd)

	rootCmd.AddCommand(serveCmd)

	// Shadow global flags
	shadowGlobalCommandFlags(serveCmd)
}
