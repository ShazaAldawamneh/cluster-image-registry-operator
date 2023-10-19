package azure

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"

	"github.com/openshift/cluster-image-registry-operator/pkg/metrics"
)

// cacheExpiration is the cache expiration duration in minutes.
const cacheExpiration time.Duration = 20 * time.Minute

// primaryKey keeps account primary key in a cache.
var primaryKey cachedKey

// cachedKey holds an API access key in memory for five minutes.
type cachedKey struct {
	mtx           sync.Mutex
	resourceGroup string
	account       string
	value         string
	expire        time.Time
}

// get returns the cached key if it is not expired yet, if expired fetches the key
// remotely using provided AccountsClient.
func (k *cachedKey) get(
	ctx context.Context, cli storage.AccountsClient, resourceGroup, account string,
) (string, error) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	if k.resourceGroup == resourceGroup && k.account == account && time.Now().Before(k.expire) {
		metrics.AzureKeyCacheHit()
		return k.value, nil
	}
	metrics.AzureKeyCacheMiss()

	keysResponse, err := cli.ListKeys(ctx, resourceGroup, account, storage.Kerb)
	if err != nil {
		return "", err
	}

	k.resourceGroup = resourceGroup
	k.account = account
	k.value = *(*keysResponse.Keys)[0].Value
	k.expire = time.Now().Add(cacheExpiration)
	return k.value, nil
}
