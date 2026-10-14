ALTER TABLE users DROP CONSTRAINT chk_users_role;
UPDATE users SET role = 'member' WHERE role = 'reader';
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';
ALTER TABLE users ADD CONSTRAINT chk_users_role CHECK (role IN ('member', 'librarian', 'admin'));
ALTER TABLE users DROP COLUMN password_hash;
