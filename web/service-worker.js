const fallbackCachePrefix = 'wade-server-unavailable';
const fallbackPageVersion = '__FALLBACK_PAGE_VERSION__';
const fallbackCacheName = `${fallbackCachePrefix}-${fallbackPageVersion}`;
const fallbackPageURL = `/static/server-unavailable.html?v=${fallbackPageVersion}`;

self.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const fallbackCache = await caches.open(fallbackCacheName);

      await fallbackCache.add(new Request(fallbackPageURL, { cache: 'reload' }));
      await self.skipWaiting();
    })()
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const cacheNames = await caches.keys();
      const staleCacheNames = cacheNames.filter(
        (cacheName) => cacheName.startsWith(fallbackCachePrefix) && cacheName !== fallbackCacheName
      );

      await Promise.all(staleCacheNames.map((cacheName) => caches.delete(cacheName)));
      await self.clients.claim();
    })()
  );
});

self.addEventListener('fetch', (event) => {
  if (event.request.mode !== 'navigate') {
    return;
  }

  event.respondWith(
    (async () => {
      try {
        return await fetch(event.request, { cache: 'no-store' });
      } catch {
        const fallbackCache = await caches.open(fallbackCacheName);
        const fallbackPage = await fallbackCache.match(fallbackPageURL);

        return fallbackPage ?? Response.error();
      }
    })()
  );
});
