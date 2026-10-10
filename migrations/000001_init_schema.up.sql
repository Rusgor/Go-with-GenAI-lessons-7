CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE authors (
    id BIGSERIAL PRIMARY KEY,
    full_name TEXT NOT NULL,
    bio TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_authors_full_name ON authors (full_name);

CREATE TABLE books (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    isbn TEXT NOT NULL UNIQUE,
    category_id BIGINT NOT NULL,
    published_year INTEGER,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_books_category FOREIGN KEY (category_id)
        REFERENCES categories (id) ON DELETE RESTRICT
);
CREATE INDEX idx_books_category_id ON books (category_id);

CREATE TABLE book_authors (
    book_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    PRIMARY KEY (book_id, author_id),
    CONSTRAINT fk_book_authors_book FOREIGN KEY (book_id)
        REFERENCES books (id) ON DELETE CASCADE,
    CONSTRAINT fk_book_authors_author FOREIGN KEY (author_id)
        REFERENCES authors (id) ON DELETE RESTRICT
);
CREATE INDEX idx_book_authors_author_id ON book_authors (author_id);

CREATE TABLE copies (
    id BIGSERIAL PRIMARY KEY,
    book_id BIGINT NOT NULL,
    inventory_no TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_copies_status CHECK (status IN ('available', 'checked_out', 'lost')),
    CONSTRAINT fk_copies_book FOREIGN KEY (book_id)
        REFERENCES books (id) ON DELETE CASCADE
);
CREATE INDEX idx_copies_book_id ON copies (book_id);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_users_role CHECK (role IN ('member', 'librarian', 'admin'))
);
