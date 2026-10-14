ALTER TABLE users ADD COLUMN password_hash TEXT;

UPDATE users SET role = 'reader' WHERE role = 'member';
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'reader';
ALTER TABLE users DROP CONSTRAINT chk_users_role;
ALTER TABLE users ADD CONSTRAINT chk_users_role CHECK (role IN ('reader', 'librarian', 'admin'));
