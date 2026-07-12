DROP INDEX IF EXISTS idx_articles_cover_media_id;
ALTER TABLE articles DROP COLUMN IF EXISTS cover_media_id;
DROP TABLE IF EXISTS media_translations;
DROP TABLE IF EXISTS media_assets;
