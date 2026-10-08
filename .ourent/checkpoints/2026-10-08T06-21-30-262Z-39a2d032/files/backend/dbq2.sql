SELECT id, storage_key, url, is_private, created_at FROM media ORDER BY id DESC LIMIT 10;
SELECT key, value FROM site_settings WHERE key IN ('hero_title','hero_description') ORDER BY key;
