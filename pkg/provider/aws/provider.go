// pkg/providers/aws/provider.go
//
// Real AWS provider for Orkestra.
//
// Handles the "aws" block in Katalog declarations.
// Uses the official AWS SDK v2 — no mocks, no stubs.
//
// Supported resource kinds:
//
//	s3      — S3 buckets (create, versioning, tagging, delete)
//	rds     — RDS instances (create, modify, delete, status polling)
//	route53 — Route53 records (upsert, delete)
//
// Installation:
//
//	go get github.com/aws/aws-sdk-go-v2
//	go get github.com/aws/aws-sdk-go-v2/config
//	go get github.com/aws/aws-sdk-go-v2/service/s3
//	go get github.com/aws/aws-sdk-go-v2/service/rds
//	go get github.com/aws/aws-sdk-go-v2/service/route53
//
// Registration:
//
//	cfg, _ := awsconfig.LoadDefaultConfig(ctx)
//	registry.Register(awsprovider.New(cfg))
//
// Credentials:
//
//	Standard AWS credential chain — environment variables, ~/.aws/credentials,
//	EC2 instance profile, ECS task role. No special Orkestra configuration needed.
//	For per-CR credentials, specify credentials.secretName in the declaration.
//
// Katalog:
//
//	providers:
//	  aws:
//	    - s3:
//	        bucket: "{{ .metadata.name }}-{{ .metadata.namespace }}"
//	        region: "{{ .spec.region }}"
//	        versioning: "true"
//
//	    - rds:
//	        identifier: "{{ .metadata.name }}-db"
//	        instanceClass: db.t3.micro
//	        engine: postgres
//	        engineVersion: "15"
//	        storage: "20"
//	        multiAZ: "false"
//	        credentials:
//	          secretName: my-aws-creds   # optional: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
//	        when:
//	          - field: spec.needsDatabase
//	            equals: "true"
package awsprovider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

// Provider implements orktypes.Provider for the "aws" block.
type Provider struct {
	cfg aws.Config // base config — overridden per-declaration if credentials.secretName is set
}

// New creates an AWS provider using the default credential chain.
// The AWS SDK resolves credentials in standard order:
//  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//  2. ~/.aws/credentials
//  3. EC2 instance profile / ECS task role
func New(cfg aws.Config) *Provider {
	return &Provider{cfg: cfg}
}

// NewFromContext loads config from the ambient environment and returns a Provider.
// Convenience constructor for operators that do not manage the aws.Config directly.
func NewFromContext(ctx context.Context) (*Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws: loading default config: %w", err)
	}
	return New(cfg), nil
}

// NewFromAuth creates an AWS provider using explicit credentials from a map.
// Recognised keys: "accessKeyId", "secretAccessKey", "sessionToken", "region".
// Any missing key falls back to the standard AWS credential chain.
// Call this when credentials come from the Katalog providers[].auth block.
func NewFromAuth(ctx context.Context, auth map[string]string) (*Provider, error) {
	opts := []func(*awsconfig.LoadOptions) error{}

	keyID := auth["accessKeyId"]
	secret := auth["secretAccessKey"]
	token := auth["sessionToken"] // optional

	if keyID != "" && secret != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(keyID, secret, token),
		))
	}
	if region := auth["region"]; region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws: loading config from auth: %w", err)
	}
	return New(cfg), nil
}

func (p *Provider) Name() string { return "aws" }

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
	for _, decl := range req.Declarations {
		cfg, err := p.resolveConfig(ctx, req, decl)
		if err != nil {
			return fmt.Errorf("aws.%s: resolving credentials: %w", decl.Kind, err)
		}

		switch decl.Kind {
		case "s3":
			if err := p.reconcileS3(ctx, cfg, req, decl); err != nil {
				return fmt.Errorf("aws.s3: %w", err)
			}
		case "rds":
			if err := p.reconcileRDS(ctx, cfg, req, decl); err != nil {
				return fmt.Errorf("aws.rds: %w", err)
			}
		case "route53":
			if err := p.reconcileRoute53(ctx, cfg, req, decl); err != nil {
				return fmt.Errorf("aws.route53: %w", err)
			}
		default:
			req.Logger.Warn().
				Str("kind", decl.Kind).
				Msg("aws: unknown resource kind — skipped")
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
	for _, decl := range req.Declarations {
		cfg, err := p.resolveConfig(ctx, orktypes.ReconcileRequest{
			Kube:           req.Kube,
			Logger:         req.Logger,
			OwnerName:      req.OwnerName,
			OwnerNamespace: req.OwnerNamespace,
		}, decl)
		if err != nil {
			return fmt.Errorf("aws.%s: resolving credentials: %w", decl.Kind, err)
		}

		switch decl.Kind {
		case "s3":
			if err := p.deleteS3(ctx, cfg, req, decl); err != nil {
				return fmt.Errorf("aws.s3 delete: %w", err)
			}
		case "rds":
			if err := p.deleteRDS(ctx, cfg, req, decl); err != nil {
				return fmt.Errorf("aws.rds delete: %w", err)
			}
		case "route53":
			if err := p.deleteRoute53(ctx, cfg, req, decl); err != nil {
				return fmt.Errorf("aws.route53 delete: %w", err)
			}
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// S3
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileS3(ctx context.Context, cfg aws.Config, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	bucket, err := decl.Require("bucket")
	if err != nil {
		return err
	}
	region := decl.Field("region", cfg.Region)
	versioning := decl.Field("versioning", "false") == "true"

	// S3 buckets are regional — use a region-specific client
	regionalCfg := cfg.Copy()
	regionalCfg.Region = region
	client := s3.NewFromConfig(regionalCfg)

	// Check existence — HeadBucket returns 404 if absent
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		if !isS3NotFound(err) {
			return fmt.Errorf("checking bucket %q: %w", bucket, err)
		}

		// Create
		input := &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		}
		// us-east-1 does not accept a LocationConstraint
		if region != "us-east-1" {
			input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
				LocationConstraint: s3types.BucketLocationConstraint(region),
			}
		}
		if _, err := client.CreateBucket(ctx, input); err != nil {
			return fmt.Errorf("creating bucket %q: %w", bucket, err)
		}
		req.Logger.Info().
			Str("bucket", bucket).
			Str("region", region).
			Msg("aws: S3 bucket created")
	}

	// Tag with Orkestra ownership — enables external audit
	if _, err := client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &s3types.Tagging{
			TagSet: []s3types.Tag{
				{Key: aws.String("orkestra-owner"), Value: aws.String(req.OwnerName)},
				{Key: aws.String("orkestra-namespace"), Value: aws.String(req.OwnerNamespace)},
			},
		},
	}); err != nil {
		req.Logger.Warn().Err(err).Str("bucket", bucket).Msg("aws: could not tag bucket — continuing")
	}

	// Versioning — apply desired state regardless of current
	vStatus := s3types.BucketVersioningStatusSuspended
	if versioning {
		vStatus = s3types.BucketVersioningStatusEnabled
	}
	if _, err := client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: vStatus,
		},
	}); err != nil {
		return fmt.Errorf("setting versioning on %q: %w", bucket, err)
	}

	req.Logger.Debug().
		Str("bucket", bucket).
		Bool("versioning", versioning).
		Msg("aws: S3 bucket reconciled")

	return nil
}

func (p *Provider) deleteS3(ctx context.Context, cfg aws.Config, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	bucket := decl.Field("bucket", "")
	if bucket == "" {
		return nil
	}
	region := decl.Field("region", cfg.Region)

	regionalCfg := cfg.Copy()
	regionalCfg.Region = region
	client := s3.NewFromConfig(regionalCfg)

	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		if isS3NotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("checking bucket %q before delete: %w", bucket, err)
	}

	// Note: bucket must be empty before deletion.
	// For production providers, implement object deletion here.
	// This provider deletes only empty buckets.
	if _, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}); err != nil {
		return fmt.Errorf("deleting bucket %q: %w", bucket, err)
	}

	req.Logger.Info().Str("bucket", bucket).Msg("aws: S3 bucket deleted")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RDS
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileRDS(ctx context.Context, cfg aws.Config, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	identifier, err := decl.Require("identifier")
	if err != nil {
		// Default to owner name + "-db" for convenience
		identifier = req.OwnerName + "-db"
	}

	instanceClass, err := decl.Require("instanceClass")
	if err != nil {
		return err
	}

	engine := decl.Field("engine", "postgres")
	engineVersion := decl.Field("engineVersion", "15")
	storage := int32(20)
	if s := decl.Field("storage", ""); s != "" {
		if n, err := strconv.ParseInt(s, 10, 32); err == nil {
			storage = int32(n)
		}
	}
	multiAZ := decl.Field("multiAZ", "false") == "true"

	client := rds.NewFromConfig(cfg)

	// Check existence
	out, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(identifier),
	})
	if err != nil {
		if !isRDSNotFound(err) {
			return fmt.Errorf("describing RDS instance %q: %w", identifier, err)
		}

		// Create
		if _, err := client.CreateDBInstance(ctx, &rds.CreateDBInstanceInput{
			DBInstanceIdentifier: aws.String(identifier),
			DBInstanceClass:      aws.String(instanceClass),
			Engine:               aws.String(engine),
			EngineVersion:        aws.String(engineVersion),
			AllocatedStorage:     aws.Int32(storage),
			MultiAZ:              aws.Bool(multiAZ),
			// MasterUsername and MasterUserPassword would come from a Secret
			// in a full implementation — omitted here for clarity
			Tags: []rdstypes.Tag{
				{Key: aws.String("orkestra-owner"), Value: aws.String(req.OwnerName)},
				{Key: aws.String("orkestra-namespace"), Value: aws.String(req.OwnerNamespace)},
			},
		}); err != nil {
			return fmt.Errorf("creating RDS instance %q: %w", identifier, err)
		}

		req.Logger.Info().
			Str("identifier", identifier).
			Str("engine", engine).
			Str("class", instanceClass).
			Msg("aws: RDS instance creation initiated — will be available in ~10 minutes")

		// Return nil — not an error. The instance is provisioning.
		// Check status on next reconcile cycle.
		return nil
	}

	if len(out.DBInstances) == 0 {
		return nil
	}

	instance := out.DBInstances[0]

	// Still creating — check again next cycle
	status := aws.ToString(instance.DBInstanceStatus)
	if status == "creating" || status == "modifying" || status == "backing-up" {
		req.Logger.Info().
			Str("identifier", identifier).
			Str("status", status).
			Msg("aws: RDS instance not yet available — will check next reconcile")
		return nil
	}

	if status != "available" {
		req.Logger.Warn().
			Str("identifier", identifier).
			Str("status", status).
			Msg("aws: RDS instance in unexpected state")
		return nil
	}

	// Drift correction — instanceClass and multiAZ can be modified
	needsModify := false
	if aws.ToString(instance.DBInstanceClass) != instanceClass {
		needsModify = true
	}
	if aws.ToBool(instance.MultiAZ) != multiAZ {
		needsModify = true
	}

	if needsModify {
		if _, err := client.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
			DBInstanceIdentifier: aws.String(identifier),
			DBInstanceClass:      aws.String(instanceClass),
			MultiAZ:              aws.Bool(multiAZ),
			ApplyImmediately:     aws.Bool(false), // apply during next maintenance window
		}); err != nil {
			return fmt.Errorf("modifying RDS instance %q: %w", identifier, err)
		}
		req.Logger.Info().
			Str("identifier", identifier).
			Str("class", instanceClass).
			Msg("aws: RDS instance modification scheduled")
	}

	req.Logger.Debug().
		Str("identifier", identifier).
		Str("status", status).
		Str("endpoint", aws.ToString(instance.Endpoint.Address)).
		Msg("aws: RDS instance reconciled")

	return nil
}

func (p *Provider) deleteRDS(ctx context.Context, cfg aws.Config, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	identifier := decl.Field("identifier", req.OwnerName+"-db")
	client := rds.NewFromConfig(cfg)

	out, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(identifier),
	})
	if err != nil {
		if isRDSNotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("describing RDS instance %q: %w", identifier, err)
	}

	if len(out.DBInstances) == 0 {
		return nil
	}

	status := aws.ToString(out.DBInstances[0].DBInstanceStatus)
	if status == "deleting" {
		req.Logger.Info().Str("identifier", identifier).Msg("aws: RDS instance already deleting")
		return nil
	}

	if _, err := client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   aws.String(identifier),
		SkipFinalSnapshot:      aws.Bool(true),
		DeleteAutomatedBackups: aws.Bool(false),
	}); err != nil {
		return fmt.Errorf("deleting RDS instance %q: %w", identifier, err)
	}

	req.Logger.Info().Str("identifier", identifier).Msg("aws: RDS instance deletion initiated")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Route53
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileRoute53(ctx context.Context, cfg aws.Config, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	zoneID, err := decl.Require("zoneId")
	if err != nil {
		return err
	}
	record, err := decl.Require("record")
	if err != nil {
		return err
	}
	target := decl.Field("target", "")
	if target == "" {
		req.Logger.Info().Str("record", record).Msg("aws: Route53 target not yet available — will retry")
		return nil
	}

	recordType := decl.Field("type", "CNAME")
	ttl := int64(300)
	if t := decl.Field("ttl", ""); t != "" {
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			ttl = n
		}
	}

	client := route53.NewFromConfig(cfg)

	if _, err := client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionUpsert,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: aws.String(record),
						Type: r53types.RRType(recordType),
						TTL:  aws.Int64(ttl),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: aws.String(target)},
						},
					},
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("upserting Route53 record %q: %w", record, err)
	}

	req.Logger.Info().
		Str("record", record).
		Str("target", target).
		Msg("aws: Route53 record upserted")

	return nil
}

func (p *Provider) deleteRoute53(ctx context.Context, cfg aws.Config, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	zoneID := decl.Field("zoneId", "")
	record := decl.Field("record", "")
	target := decl.Field("target", "")

	if zoneID == "" || record == "" || target == "" {
		return nil
	}

	recordType := decl.Field("type", "CNAME")
	client := route53.NewFromConfig(cfg)

	if _, err := client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{
				{
					Action: r53types.ChangeActionDelete,
					ResourceRecordSet: &r53types.ResourceRecordSet{
						Name: aws.String(record),
						Type: r53types.RRType(recordType),
						TTL:  aws.Int64(300),
						ResourceRecords: []r53types.ResourceRecord{
							{Value: aws.String(target)},
						},
					},
				},
			},
		},
	}); err != nil {
		// Record not found — already deleted
		if isRoute53NotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting Route53 record %q: %w", record, err)
	}

	req.Logger.Info().Str("record", record).Msg("aws: Route53 record deleted")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Credential override
// ─────────────────────────────────────────────────────────────────────────────

// resolveConfig returns a cfg optionally overridden with per-declaration credentials.
// If credentials.secretName is set, reads AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
// from the named Kubernetes Secret and uses them instead of the ambient credentials.
func (p *Provider) resolveConfig(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) (aws.Config, error) {
	secretName := decl.Field("credentials.secretName", "")
	if secretName == "" {
		return p.cfg, nil
	}

	data, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, secretName)
	if err != nil {
		return aws.Config{}, fmt.Errorf("reading secret %q: %w", secretName, err)
	}

	accessKey := string(data["AWS_ACCESS_KEY_ID"])
	secretKey := string(data["AWS_SECRET_ACCESS_KEY"])
	sessionToken := string(data["AWS_SESSION_TOKEN"])

	if accessKey == "" || secretKey == "" {
		return aws.Config{}, fmt.Errorf("secret %q missing AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY", secretName)
	}

	cfg := p.cfg.Copy()
	cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)
	return cfg, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Error helpers
// ─────────────────────────────────────────────────────────────────────────────

func isS3NotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchBucket" || apiErr.ErrorCode() == "NotFound"
	}
	return false
}

func isRDSNotFound(err error) bool {
	var notFound *rdstypes.DBInstanceNotFoundFault
	return errors.As(err, &notFound)
}

func isRoute53NotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchHostedZone" || apiErr.ErrorCode() == "InvalidChangeBatch"
	}
	return false
}
