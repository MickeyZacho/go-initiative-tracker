-- +goose Up
-- A user's personal look (colours, radius, fonts, custom CSS) from the theme
-- panel, as the frontend's ThemeTokens JSON. NULL means the app default, so
-- changes to the default still reach users who never customised anything.
ALTER TABLE users ADD COLUMN IF NOT EXISTS theme JSONB;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS theme;
