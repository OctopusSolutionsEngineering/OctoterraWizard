package sensitivevariables

import "testing"
import _ "github.com/microsoft/go-mssqldb"

func TestExtractVariables(t *testing.T) {
	return

	result, err := ExtractVariables("localhost", "1433", "Octopus", "SA", "Password01!", "6EdU6IWsCtMEwk0kPKflQQ==")

	if err != nil {
		t.Fatalf("Failed to extract variables: %v", err)
	}

	if result != "e647abb66147e2d701c0b44934063b6e27dd4af84d10145d196a4a50bffcfc14_sensitive_value = \"success\"\n" {
		t.Errorf("Expected %s, got %s", "success", result)
	}
}
