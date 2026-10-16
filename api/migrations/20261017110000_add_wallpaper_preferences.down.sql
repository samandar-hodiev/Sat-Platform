BEGIN;

ALTER TABLE user_preferences
    DROP COLUMN wallpaper_light_source,
    DROP COLUMN wallpaper_light_gallery_id,
    DROP COLUMN wallpaper_light_key,
    DROP COLUMN wallpaper_dark_source,
    DROP COLUMN wallpaper_dark_gallery_id,
    DROP COLUMN wallpaper_dark_key,
    DROP COLUMN wallpaper_opacity,
    DROP COLUMN wallpaper_blur;

COMMIT;
