-- +goose Up
-- Создаем системную команду админов
INSERT INTO teams (team_name, created_at)
VALUES ('admins', NOW())
    ON CONFLICT (team_name) DO NOTHING;

-- Создаем первого главного администратора
INSERT INTO users (user_id, username, team_name, is_active, created_at, updated_at)
VALUES ('admin', 'System Admin', 'admins', true, NOW(), NOW())
    ON CONFLICT (user_id) DO UPDATE SET
    username = EXCLUDED.username,
                                 team_name = EXCLUDED.team_name,
                                 is_active = EXCLUDED.is_active,
                                 updated_at = NOW();

-- Создаем индекс для быстрой проверки админов
CREATE INDEX IF NOT EXISTS idx_users_team_admins ON users(team_name) WHERE team_name = 'admins';

-- +goose Down
-- Удаляем админа
DELETE FROM users WHERE user_id = 'admin';

-- Удаляем команду админов (каскадно удалит всех пользователей команды)
DELETE FROM teams WHERE team_name = 'admins';

-- Удаляем индекс
DROP INDEX IF EXISTS idx_users_team_admins;