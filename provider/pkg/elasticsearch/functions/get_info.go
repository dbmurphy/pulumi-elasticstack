package functions

import (
	"context"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// GetInfo retrieves Elasticsearch cluster information via GET /.
type GetInfo struct{}

// GetInfoArgs holds the (empty) input arguments for getInfo.
type GetInfoArgs struct{}

// GetInfoResult holds the output of the getInfo function.
type GetInfoResult struct {
	Name                             string `pulumi:"name"`
	ClusterName                      string `pulumi:"clusterName"`
	ClusterUUID                      string `pulumi:"clusterUuid"`
	VersionNumber                    string `pulumi:"versionNumber"`
	BuildFlavor                      string `pulumi:"buildFlavor"`
	BuildType                        string `pulumi:"buildType"`
	BuildHash                        string `pulumi:"buildHash"`
	BuildDate                        string `pulumi:"buildDate"`
	BuildSnapshot                    bool   `pulumi:"buildSnapshot"`
	LuceneVersion                    string `pulumi:"luceneVersion"`
	MinimumWireCompatibilityVersion  string `pulumi:"minimumWireCompatibilityVersion"`
	MinimumIndexCompatibilityVersion string `pulumi:"minimumIndexCompatibilityVersion"`
	Tagline                          string `pulumi:"tagline"`
}

// Annotate ...
func (f *GetInfo) Annotate(a infer.Annotator) {
	a.Describe(f, "Get Elasticsearch cluster information (version, name, UUID).")
	a.SetToken("elasticsearch", "getInfo")
}

// Annotate ...
func (f *GetInfoArgs) Annotate(a infer.Annotator) {
	// No input arguments needed
}

// Annotate ...
func (f *GetInfoResult) Annotate(a infer.Annotator) {
	a.Describe(&f.Name, "The node name.")
	a.Describe(&f.ClusterName, "The cluster name.")
	a.Describe(&f.ClusterUUID, "The cluster UUID.")
	a.Describe(&f.VersionNumber, "The Elasticsearch version number.")
	a.Describe(&f.BuildFlavor, "The build flavor (default, oss).")
	a.Describe(&f.BuildType, "The build type (tar, docker, etc.).")
	a.Describe(&f.BuildHash, "The build hash.")
	a.Describe(&f.BuildDate, "The build date.")
	a.Describe(&f.BuildSnapshot, "Whether this is a snapshot build.")
	a.Describe(&f.LuceneVersion, "The Lucene version.")
	a.Describe(&f.MinimumWireCompatibilityVersion, "The minimum wire protocol version this node is compatible with.")
	a.Describe(&f.MinimumIndexCompatibilityVersion, "The minimum index version this node can read.")
	a.Describe(&f.Tagline, "The Elasticsearch tagline.")
}

// Invoke ...
func (*GetInfo) Invoke(
	ctx context.Context,
	req infer.FunctionRequest[GetInfoArgs],
) (infer.FunctionResponse[GetInfoResult], error) {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.FunctionResponse[GetInfoResult]{}, err
	}

	info, err := esClient.GetClusterInfo(ctx)
	if err != nil {
		return infer.FunctionResponse[GetInfoResult]{}, err
	}

	return infer.FunctionResponse[GetInfoResult]{
		Output: GetInfoResult{
			Name:                             info.Name,
			ClusterName:                      info.ClusterName,
			ClusterUUID:                      info.ClusterUUID,
			VersionNumber:                    info.Version.Number,
			BuildFlavor:                      info.Version.BuildFlavor,
			BuildType:                        info.Version.BuildType,
			BuildHash:                        info.Version.BuildHash,
			BuildDate:                        info.Version.BuildDate,
			BuildSnapshot:                    info.Version.BuildSnapshot,
			LuceneVersion:                    info.Version.LuceneVersion,
			MinimumWireCompatibilityVersion:  info.Version.MinimumWireCompatibilityVersion,
			MinimumIndexCompatibilityVersion: info.Version.MinimumIndexCompatibilityVersion,
			Tagline:                          info.Tagline,
		},
	}, nil
}
