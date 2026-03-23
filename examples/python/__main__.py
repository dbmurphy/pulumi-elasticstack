import json
import pulumi
import pulumi_elasticstack as elasticstack

# Elasticsearch index
my_index = elasticstack.elasticsearch.Index("my-index",
    name="my-application-logs",
    number_of_shards=1,
    number_of_replicas=1,
    adopt_on_create=True,
    deletion_protection=True,
)

# Index template
logs_template = elasticstack.elasticsearch.IndexTemplate("logs-template",
    name="my-logs-template",
    index_patterns=["my-logs-*"],
    template=json.dumps({
        "settings": {"number_of_shards": 1},
        "mappings": {
            "properties": {
                "@timestamp": {"type": "date"},
                "message": {"type": "text"},
            }
        },
    }),
    priority=100,
)

# ILM policy
ilm_policy = elasticstack.elasticsearch.IndexLifecycle("logs-ilm",
    name="logs-lifecycle",
    hot_phase=json.dumps({
        "actions": {"rollover": {"max_age": "7d"}},
    }),
    delete_phase=json.dumps({
        "min_age": "30d",
        "actions": {"delete": {}},
    }),
)

# Kibana space
dev_space = elasticstack.kibana.Space("dev-space",
    space_id="development",
    name="Development",
    description="Development team workspace",
)

# Fleet agent policy
agent_policy = elasticstack.fleet.AgentPolicy("monitoring-policy",
    name="monitoring-agents",
    namespace="default",
    monitor_logs=True,
    monitor_metrics=True,
)

pulumi.export("index_name", my_index.name)
pulumi.export("space_name", dev_space.name)
