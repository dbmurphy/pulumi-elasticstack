// Package main is the entry point for the pulumi-resource-elasticstack provider.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/apm"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/cloud"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/cluster"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/enrich"
	esfunctions "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/functions"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/index"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/ingest"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/lifecycle"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/logstash"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/ml"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/script"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/security"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/snapshot"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/template"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/transform"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/elasticsearch/watcher"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/fleet"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/alerting"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/dashboard"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/dataview"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/detection"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/savedobj"
	kbsecurity "github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/security"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/slo"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/space"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/kibana/synthetics"
	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

func main() {
	p, err := provider.NewProvider(
		// Resources
		[]infer.InferredResource{
			// Cluster
			infer.Resource[
				*cluster.Settings,
				cluster.SettingsInputs,
				cluster.SettingsState,
			](&cluster.Settings{}),
			// Index
			infer.Resource[
				*index.Index, index.Inputs, index.State,
			](&index.Index{}),
			infer.Resource[
				*index.AliasResource,
				index.AliasInputs,
				index.AliasState,
			](&index.AliasResource{}),
			infer.Resource[
				*index.DataStream,
				index.DataStreamInputs,
				index.DataStreamState,
			](&index.DataStream{}),
			infer.Resource[
				*index.DataStreamLifecycle,
				index.DataStreamLifecycleInputs,
				index.DataStreamLifecycleState,
			](&index.DataStreamLifecycle{}),
			// Templates
			infer.Resource[
				*template.IndexTemplate,
				template.IndexTemplateInputs,
				template.IndexTemplateState,
			](&template.IndexTemplate{}),
			infer.Resource[
				*template.ComponentTemplate,
				template.ComponentTemplateInputs,
				template.ComponentTemplateState,
			](&template.ComponentTemplate{}),
			infer.Resource[
				*template.IndexTemplateIlmAttachment,
				template.IndexTemplateIlmAttachmentInputs,
				template.IndexTemplateIlmAttachmentState,
			](&template.IndexTemplateIlmAttachment{}),
			// Lifecycle
			infer.Resource[
				*lifecycle.IndexLifecycle,
				lifecycle.IndexLifecycleInputs,
				lifecycle.IndexLifecycleState,
			](&lifecycle.IndexLifecycle{}),
			// Snapshot
			infer.Resource[
				*snapshot.Lifecycle,
				snapshot.LifecycleInputs,
				snapshot.LifecycleState,
			](&snapshot.Lifecycle{}),
			infer.Resource[
				*snapshot.Repository,
				snapshot.RepositoryInputs,
				snapshot.RepositoryState,
			](&snapshot.Repository{}),
			// Ingest
			infer.Resource[
				*ingest.Pipeline,
				ingest.PipelineInputs,
				ingest.PipelineState,
			](&ingest.Pipeline{}),
			// Security
			infer.Resource[
				*security.User,
				security.UserInputs,
				security.UserState,
			](&security.User{}),
			infer.Resource[
				*security.SystemUser,
				security.SystemUserInputs,
				security.SystemUserState,
			](&security.SystemUser{}),
			infer.Resource[
				*security.Role,
				security.RoleInputs,
				security.RoleState,
			](&security.Role{}),
			infer.Resource[
				*security.RoleMapping,
				security.RoleMappingInputs,
				security.RoleMappingState,
			](&security.RoleMapping{}),
			infer.Resource[
				*security.ApiKey,
				security.ApiKeyInputs,
				security.ApiKeyState,
			](&security.ApiKey{}),
			// ML
			infer.Resource[
				*ml.AnomalyDetectionJob,
				ml.AnomalyDetectionJobInputs,
				ml.AnomalyDetectionJobState,
			](&ml.AnomalyDetectionJob{}),
			infer.Resource[
				*ml.Datafeed,
				ml.DatafeedInputs,
				ml.DatafeedState,
			](&ml.Datafeed{}),
			infer.Resource[
				*ml.DatafeedStateControl,
				ml.DatafeedStateControlInputs,
				ml.DatafeedStateControlState,
			](&ml.DatafeedStateControl{}),
			infer.Resource[
				*ml.JobStateControl,
				ml.JobStateControlInputs,
				ml.JobStateControlState,
			](&ml.JobStateControl{}),
			// Transform
			infer.Resource[
				*transform.Transform,
				transform.Inputs,
				transform.State,
			](&transform.Transform{}),
			// Watcher
			infer.Resource[
				*watcher.Watch,
				watcher.WatchInputs,
				watcher.WatchState,
			](&watcher.Watch{}),
			// Enrich
			infer.Resource[
				*enrich.Policy,
				enrich.PolicyInputs,
				enrich.PolicyState,
			](&enrich.Policy{}),
			// Script
			infer.Resource[
				*script.Script,
				script.Inputs,
				script.State,
			](&script.Script{}),
			// Logstash
			infer.Resource[
				*logstash.Pipeline,
				logstash.PipelineInputs,
				logstash.PipelineState,
			](&logstash.Pipeline{}),

			// --- Kibana Resources ---
			// Space
			infer.Resource[
				*space.Space, space.Inputs, space.State,
			](&space.Space{}),
			// Kibana Security
			infer.Resource[
				*kbsecurity.KibanaSecurityRole,
				kbsecurity.KibanaSecurityRoleInputs,
				kbsecurity.KibanaSecurityRoleState,
			](&kbsecurity.KibanaSecurityRole{}),
			// Alerting
			infer.Resource[
				*alerting.ActionConnector,
				alerting.ActionConnectorInputs,
				alerting.ActionConnectorState,
			](&alerting.ActionConnector{}),
			infer.Resource[
				*alerting.Rule,
				alerting.RuleInputs,
				alerting.RuleState,
			](&alerting.Rule{}),
			infer.Resource[
				*alerting.MaintenanceWindow,
				alerting.MaintenanceWindowInputs,
				alerting.MaintenanceWindowState,
			](&alerting.MaintenanceWindow{}),
			// Data Views
			infer.Resource[
				*dataview.DataView,
				dataview.Inputs,
				dataview.State,
			](&dataview.DataView{}),
			infer.Resource[
				*dataview.DefaultDataView,
				dataview.DefaultInputs,
				dataview.DefaultState,
			](&dataview.DefaultDataView{}),
			// Saved Objects
			infer.Resource[
				*savedobj.ImportSavedObjects,
				savedobj.ImportSavedObjectsInputs,
				savedobj.ImportSavedObjectsState,
			](&savedobj.ImportSavedObjects{}),
			// SLO
			infer.Resource[
				*slo.Slo, slo.Inputs, slo.State,
			](&slo.Slo{}),
			// Security Detection
			infer.Resource[
				*detection.SecurityDetectionRule,
				detection.SecurityDetectionRuleInputs,
				detection.SecurityDetectionRuleState,
			](&detection.SecurityDetectionRule{}),
			infer.Resource[
				*detection.SecurityEnableRule,
				detection.SecurityEnableRuleInputs,
				detection.SecurityEnableRuleState,
			](&detection.SecurityEnableRule{}),
			infer.Resource[
				*detection.InstallPrebuiltRules,
				detection.InstallPrebuiltRulesInputs,
				detection.InstallPrebuiltRulesState,
			](&detection.InstallPrebuiltRules{}),
			infer.Resource[
				*detection.SecurityExceptionList,
				detection.SecurityExceptionListInputs,
				detection.SecurityExceptionListState,
			](&detection.SecurityExceptionList{}),
			infer.Resource[
				*detection.SecurityExceptionItem,
				detection.SecurityExceptionItemInputs,
				detection.SecurityExceptionItemState,
			](&detection.SecurityExceptionItem{}),
			infer.Resource[
				*detection.SecurityList,
				detection.SecurityListInputs,
				detection.SecurityListState,
			](&detection.SecurityList{}),
			infer.Resource[
				*detection.SecurityListDataStreams,
				detection.SecurityListDataStreamsInputs,
				detection.SecurityListDataStreamsState,
			](&detection.SecurityListDataStreams{}),
			infer.Resource[
				*detection.SecurityListItem,
				detection.SecurityListItemInputs,
				detection.SecurityListItemState,
			](&detection.SecurityListItem{}),
			// Synthetics
			infer.Resource[
				*synthetics.Monitor,
				synthetics.MonitorInputs,
				synthetics.MonitorState,
			](&synthetics.Monitor{}),
			infer.Resource[
				*synthetics.Parameter,
				synthetics.ParameterInputs,
				synthetics.ParameterState,
			](&synthetics.Parameter{}),
			infer.Resource[
				*synthetics.PrivateLocation,
				synthetics.PrivateLocationInputs,
				synthetics.PrivateLocationState,
			](&synthetics.PrivateLocation{}),
			// Dashboard (experimental)
			infer.Resource[
				*dashboard.Dashboard,
				dashboard.Inputs,
				dashboard.State,
			](&dashboard.Dashboard{}),

			// --- Fleet Resources ---
			infer.Resource[
				*fleet.AgentPolicy,
				fleet.AgentPolicyInputs,
				fleet.AgentPolicyState,
			](&fleet.AgentPolicy{}),
			infer.Resource[
				*fleet.Integration,
				fleet.IntegrationInputs,
				fleet.IntegrationState,
			](&fleet.Integration{}),
			infer.Resource[
				*fleet.IntegrationPolicy,
				fleet.IntegrationPolicyInputs,
				fleet.IntegrationPolicyState,
			](&fleet.IntegrationPolicy{}),
			infer.Resource[
				*fleet.Output,
				fleet.OutputInputs,
				fleet.OutputState,
			](&fleet.Output{}),
			infer.Resource[
				*fleet.ServerHost,
				fleet.ServerHostInputs,
				fleet.ServerHostState,
			](&fleet.ServerHost{}),

			// --- APM Resources ---
			infer.Resource[
				*apm.AgentConfiguration,
				apm.AgentConfigurationInputs,
				apm.AgentConfigurationState,
			](&apm.AgentConfiguration{}),

			// --- Cloud Resources ---
			infer.Resource[
				*cloud.DeploymentPassword,
				cloud.DeploymentPasswordInputs,
				cloud.DeploymentPasswordState,
			](&cloud.DeploymentPassword{}),
			infer.Resource[
				*cloud.OrganizationMember,
				cloud.OrganizationMemberInputs,
				cloud.OrganizationMemberState,
			](&cloud.OrganizationMember{}),
			infer.Resource[
				*cloud.TrafficFilter,
				cloud.TrafficFilterInputs,
				cloud.TrafficFilterState,
			](&cloud.TrafficFilter{}),
			infer.Resource[
				*cloud.TrafficFilterAssociation,
				cloud.TrafficFilterAssociationInputs,
				cloud.TrafficFilterAssociationState,
			](&cloud.TrafficFilterAssociation{}),
		},
		// Functions
		[]infer.InferredFunction{
			infer.Function[
				*esfunctions.GetInfo,
				esfunctions.GetInfoArgs,
				esfunctions.GetInfoResult,
			](&esfunctions.GetInfo{}),
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building provider: %s\n", err.Error())
		os.Exit(1)
	}

	err = p.Run(context.Background(), provider.Name, provider.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running provider: %s\n", err.Error())
		os.Exit(1)
	}
}
