import { createAsyncStoragePersister } from '@tanstack/query-async-storage-persister';
import { get, set, del } from 'idb-keyval';
import { Query, QueryClient } from '@tanstack/react-query';

export const IDB_QUERY_CACHE_KEY = 'KIYOMI_QUERY_OFFLINE_CACHE';
export const CACHE_BUSTER = '1.0.0';

export const idbStorage = {
  getItem: async (key: string): Promise<string | null> => {
    const val = await get<string>(key);
    return val !== undefined ? val : null;
  },
  setItem: async (key: string, value: string): Promise<void> => {
    await set(key, value);
  },
  removeItem: async (key: string): Promise<void> => {
    await del(key);
  },
};

export const persister = createAsyncStoragePersister({
  storage: idbStorage,
  key: IDB_QUERY_CACHE_KEY,
  throttleTime: 1000,
});

export const shouldDehydrateQuery = (query: Query): boolean => {
  // Only persist successfully resolved queries
  if (query.state.status !== 'success') {
    return false;
  }

  const queryKey = query.queryKey;
  if (Array.isArray(queryKey)) {
    // Skip real-time logs
    if (queryKey[0] === 'plugins' && queryKey[1] === 'logs') {
      return false;
    }
    // Skip collision alerts and cache stats
    if (queryKey[0] === 'collisions') {
      return false;
    }
    if (queryKey[0] === 'system' && queryKey[1] === 'cache') {
      return false;
    }
  }

  return true;
};

export const clearQueryPersistence = async (queryClient?: QueryClient): Promise<void> => {
  await del(IDB_QUERY_CACHE_KEY);
  if (queryClient) {
    queryClient.clear();
  }
};
