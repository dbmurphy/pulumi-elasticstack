package template

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider"
)

// IndexTemplateIlmAttachment attaches an ILM policy to an existing index template.
type IndexTemplateIlmAttachment struct{}

// IndexTemplateIlmAttachmentInputs defines the input properties.
type IndexTemplateIlmAttachmentInputs struct {
	IndexTemplateName string  `pulumi:"indexTemplateName"`
	PolicyName        string  `pulumi:"policyName"`
	DataStream        *string `pulumi:"dataStream,optional"`
}

// IndexTemplateIlmAttachmentState defines the output state.
type IndexTemplateIlmAttachmentState struct {
	IndexTemplateIlmAttachmentInputs
}

var (
	_ infer.CustomDelete[IndexTemplateIlmAttachmentState] = (*IndexTemplateIlmAttachment)(nil)
	_ infer.CustomUpdate[
		IndexTemplateIlmAttachmentInputs, IndexTemplateIlmAttachmentState,
	] = (*IndexTemplateIlmAttachment)(nil)
)

// Annotate sets resource metadata and descriptions.
func (r *IndexTemplateIlmAttachment) Annotate(a infer.Annotator) {
	a.Describe(r, "Attaches an ILM policy to an existing index template.")
	a.SetToken("elasticsearch", "IndexTemplateIlmAttachment")
}

// Annotate sets input property descriptions and defaults.
func (i *IndexTemplateIlmAttachmentInputs) Annotate(a infer.Annotator) {
	a.Describe(&i.IndexTemplateName, "The name of the index template to attach the ILM policy to.")
	a.Describe(&i.PolicyName, "The name of the ILM policy to attach.")
	a.Describe(&i.DataStream, "Data stream name if the template is for a data stream.")
}

// Create provisions a new ILM attachment.
func (r *IndexTemplateIlmAttachment) Create(
	ctx context.Context,
	req infer.CreateRequest[IndexTemplateIlmAttachmentInputs],
) (infer.CreateResponse[IndexTemplateIlmAttachmentState], error) {
	if err := attachIlmPolicy(ctx, req.Inputs); err != nil {
		return infer.CreateResponse[IndexTemplateIlmAttachmentState]{}, err
	}

	id := req.Inputs.IndexTemplateName + "/" + req.Inputs.PolicyName
	return infer.CreateResponse[IndexTemplateIlmAttachmentState]{
		ID:     id,
		Output: IndexTemplateIlmAttachmentState{IndexTemplateIlmAttachmentInputs: req.Inputs},
	}, nil
}

// Update modifies an existing ILM attachment.
func (r *IndexTemplateIlmAttachment) Update(
	ctx context.Context,
	req infer.UpdateRequest[IndexTemplateIlmAttachmentInputs, IndexTemplateIlmAttachmentState],
) (infer.UpdateResponse[IndexTemplateIlmAttachmentState], error) {
	if err := attachIlmPolicy(ctx, req.Inputs); err != nil {
		return infer.UpdateResponse[IndexTemplateIlmAttachmentState]{}, err
	}

	return infer.UpdateResponse[IndexTemplateIlmAttachmentState]{
		Output: IndexTemplateIlmAttachmentState{IndexTemplateIlmAttachmentInputs: req.Inputs},
	}, nil
}

// Delete removes the ILM attachment.
func (r *IndexTemplateIlmAttachment) Delete(
	ctx context.Context,
	req infer.DeleteRequest[IndexTemplateIlmAttachmentState],
) (infer.DeleteResponse, error) {
	// Remove the ILM policy from the template by reading and re-writing without it
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	var result struct {
		IndexTemplates []struct {
			Name          string          `json:"name"`
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := esClient.GetJSON(ctx, "/_index_template/"+req.State.IndexTemplateName, &result); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to read index template: %w", err)
	}

	if len(result.IndexTemplates) == 0 {
		return infer.DeleteResponse{}, nil
	}

	var tmpl map[string]any
	if err := json.Unmarshal(result.IndexTemplates[0].IndexTemplate, &tmpl); err != nil {
		return infer.DeleteResponse{}, err
	}

	// Remove ILM settings from template.settings.index.lifecycle
	if template, ok := tmpl["template"].(map[string]any); ok {
		if settings, ok := template["settings"].(map[string]any); ok {
			if idx, ok := settings["index"].(map[string]any); ok {
				delete(idx, "lifecycle")
			}
		}
	}

	if err := esClient.PutJSON(ctx, "/_index_template/"+req.State.IndexTemplateName, tmpl, nil); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("failed to remove ILM attachment: %w", err)
	}

	return infer.DeleteResponse{}, nil
}

func attachIlmPolicy(ctx context.Context, inputs IndexTemplateIlmAttachmentInputs) error {
	cfg := infer.GetConfig[provider.Config](ctx)
	esClient, err := cfg.ESClient()
	if err != nil {
		return err
	}

	// Read the existing template
	var result struct {
		IndexTemplates []struct {
			Name          string          `json:"name"`
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := esClient.GetJSON(ctx, "/_index_template/"+inputs.IndexTemplateName, &result); err != nil {
		return fmt.Errorf("failed to read index template %s: %w", inputs.IndexTemplateName, err)
	}

	if len(result.IndexTemplates) == 0 {
		return fmt.Errorf("index template %s not found", inputs.IndexTemplateName)
	}

	var tmpl map[string]any
	if err := json.Unmarshal(result.IndexTemplates[0].IndexTemplate, &tmpl); err != nil {
		return err
	}

	// Ensure template.settings.index.lifecycle.name is set
	template, _ := tmpl["template"].(map[string]any)
	if template == nil {
		template = map[string]any{}
		tmpl["template"] = template
	}
	settings, _ := template["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
		template["settings"] = settings
	}
	idx, _ := settings["index"].(map[string]any)
	if idx == nil {
		idx = map[string]any{}
		settings["index"] = idx
	}
	lifecycle, _ := idx["lifecycle"].(map[string]any)
	if lifecycle == nil {
		lifecycle = map[string]any{}
		idx["lifecycle"] = lifecycle
	}
	lifecycle["name"] = inputs.PolicyName

	if err := esClient.PutJSON(ctx, "/_index_template/"+inputs.IndexTemplateName, tmpl, nil); err != nil {
		return fmt.Errorf("failed to attach ILM policy to template %s: %w", inputs.IndexTemplateName, err)
	}

	return nil
}
