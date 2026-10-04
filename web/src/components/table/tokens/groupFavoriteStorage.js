// Personal display preferences only; never a source of group permissions.
export const favoriteStorageKey = (userId) =>
  userId == null ? null : `token-group-favorites:v1:${userId}`;

export const readGroupFavorites = (storage, key) => {
  if (!key) return [];
  try {
    const value = JSON.parse(storage.getItem(key) || '[]');
    return Array.isArray(value)
      ? [...new Set(value.filter((item) => typeof item === 'string' && item))]
      : [];
  } catch {
    return [];
  }
};

export const availableFavorites = (groups, favorites) => {
  const names = new Set(favorites);
  return groups.filter((group) => names.has(group.value));
};

export const toggleGroupFavorite = (favorites, group) =>
  favorites.includes(group)
    ? favorites.filter((value) => value !== group)
    : [...favorites, group];
