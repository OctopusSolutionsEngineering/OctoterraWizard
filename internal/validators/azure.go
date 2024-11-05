package validators

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"io"
)

func AzureResourceGroupExists(tenantID, clientID, subscriptionId, clientSecret, resourceGroupName string) (bool, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create credential: %w", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionId, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create resource group client: %w", err)
	}

	pager := rgClient.NewListPager(&armresources.ResourceGroupsClientListOptions{})

	for pager.More() {
		results, err := pager.NextPage(context.Background())
		if err != nil {
			return false, fmt.Errorf("failed to get next page: %w", err)
		}

		for _, rg := range results.Value {
			if rg.Name != nil && *rg.Name == resourceGroupName {
				return true, nil
			}
		}
	}

	return false, nil
}

func AzureContainerExists(tenantID, clientID, clientSecret, accountName, containerName string) (bool, error) {
	// Create a credential using the service principal
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create credential: %w", err)
	}

	// Create a service client
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	serviceClient, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create service client: %w", err)
	}

	listContainersOptions := azblob.ListContainersOptions{
		Include: service.ListContainersInclude{},
	}
	containers := serviceClient.NewListContainersPager(&listContainersOptions)

	for containers.More() {
		results, pagerErr := containers.NextPage(context.Background())

		if pagerErr != nil {
			var authErr *azidentity.AuthenticationFailedError
			if errors.As(pagerErr, &authErr) {
				bytes, err := io.ReadAll(authErr.RawResponse.Body)
				if err != nil {
					return false, pagerErr
				}

				return false, fmt.Errorf("authentication failed: %s", string(bytes))
			}
			return false, fmt.Errorf("failed to get next page: %w", pagerErr)
		}

		for _, container := range results.ContainerItems {
			if container.Name != nil && *container.Name == containerName {
				return true, nil
			}
		}
	}

	return false, nil
}
