const CACHE_NAME = 'gpt-image-playground-v0.1.6'
const APP_SHELL = ['./manifest.webmanifest', './pwa-icon.svg']
const CACHEABLE_DESTINATIONS = new Set(['font', 'image'])

function shouldCacheRuntimeAsset(request, response) {
  if (!response.ok) return false
  const contentType = response.headers.get('content-type') || ''
  if (contentType.includes('text/html')) return false
  if (request.destination === 'script' || request.destination === 'style') return false
  return CACHEABLE_DESTINATIONS.has(request.destination)
}

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL)),
  )
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))),
    ),
  )
  self.clients.claim()
})

self.addEventListener('fetch', (event) => {
  const { request } = event

  if (request.method !== 'GET') return

  const url = new URL(request.url)
  if (url.origin !== self.location.origin) return

  if (request.mode === 'navigate') {
    event.respondWith(fetch(request, { cache: 'no-store' }))
    return
  }

  event.respondWith(
    caches.match(request).then((cached) => {
      if (cached) return cached

      return fetch(request).then((response) => {
        if (shouldCacheRuntimeAsset(request, response)) {
          const copy = response.clone()
          caches.open(CACHE_NAME).then((cache) => cache.put(request, copy))
        }
        return response
      })
    }),
  )
})
