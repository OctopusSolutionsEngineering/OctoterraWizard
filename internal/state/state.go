package state

type State struct {
	BackendType                   string
	Server                        string
	ServerExternal                string
	ApiKey                        string
	Space                         string
	DestinationServer             string
	DestinationServerExternal     string
	DestinationApiKey             string
	DestinationSpace              string
	AwsAccessKey                  string
	AwsSecretKey                  string
	AwsS3Bucket                   string
	AwsS3BucketRegion             string
	PromptForDelete               bool
	UseContainerImages            bool
	AzureResourceGroupName        string
	AzureStorageAccountName       string
	AzureContainerName            string
	AzureSubscriptionId           string
	AzureTenantId                 string
	AzureApplicationId            string
	AzurePassword                 string
	ExcludeAllLibraryVariableSets bool
	EnableVariableSpreading       bool
	EnableProjectRenaming         bool

	DatabaseServer    string
	DatabaseUser      string
	DatabasePass      string
	DatabasePort      string
	DatabaseName      string
	DatabaseMasterKey string
}

func (s State) GetExternalServer() string {
	if s.ServerExternal != "" {
		return s.ServerExternal
	}

	return s.Server
}
