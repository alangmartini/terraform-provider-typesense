package tfnames

const ProviderTypeName = "typesense"
const ProviderSource = "alanm/typesense"

const (
	ResourceCluster             = "cluster"
	ResourceClusterConfigChange = "cluster_config_change"
	ResourceCollection          = "collection"
	ResourceCollectionAlias     = "collection_alias"
	ResourceSynonym             = "synonym"
	ResourceOverride            = "override"
	ResourceStopwordsSet        = "stopwords_set"
	ResourcePreset              = "preset"
	ResourceAnalyticsRule       = "analytics_rule"
	ResourceAPIKey              = "api_key"
	ResourceNLSearchModel       = "nl_search_model"
	ResourceConversationModel   = "conversation_model"
	ResourceStemmingDictionary  = "stemming_dictionary"
)

const (
	DataSourceCollections = "collections"
	DataSourceAPIKeys     = "api_keys"
	DataSourceServerInfo  = "server_info"
)

var ResourceNames = []string{
	ResourceCluster,
	ResourceClusterConfigChange,
	ResourceCollection,
	ResourceCollectionAlias,
	ResourceSynonym,
	ResourceOverride,
	ResourceStopwordsSet,
	ResourcePreset,
	ResourceAnalyticsRule,
	ResourceAPIKey,
	ResourceNLSearchModel,
	ResourceConversationModel,
	ResourceStemmingDictionary,
}

var GeneratedResourceNames = []string{
	ResourceCluster,
	ResourceCollection,
	ResourceStopwordsSet,
	ResourceSynonym,
	ResourceOverride,
	ResourceAnalyticsRule,
	ResourceAPIKey,
	ResourceNLSearchModel,
	ResourceConversationModel,
}

var DataSourceNames = []string{
	DataSourceCollections,
	DataSourceAPIKeys,
	DataSourceServerInfo,
}

func TypeName(providerTypeName, name string) string {
	return providerTypeName + "_" + name
}

func FullTypeName(name string) string {
	return TypeName(ProviderTypeName, name)
}
