-- Legacy author values use commas as author separators. The source snapshot
-- preserves the exact legacy text for rollback, including punctuation/spacing.
CREATE TABLE book_legacy_values (
    book_id BIGINT PRIMARY KEY REFERENCES books (id) ON DELETE CASCADE,
    author_text TEXT NOT NULL,
    category_text TEXT NOT NULL
);
INSERT INTO book_legacy_values (book_id, author_text, category_text)
SELECT id, author, category FROM books;

CREATE TABLE authors (
    id BIGSERIAL PRIMARY KEY,
    full_name TEXT NOT NULL UNIQUE,
    bio TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO authors (full_name)
SELECT DISTINCT BTRIM(split.author_name)
FROM books, LATERAL unnest(string_to_array(books.author, ',')) AS split(author_name)
WHERE BTRIM(split.author_name) <> ''
ON CONFLICT (full_name) DO NOTHING;

INSERT INTO categories (name)
SELECT DISTINCT CASE WHEN BTRIM(category) = '' THEN 'Uncategorized' ELSE BTRIM(category) END
FROM books
ON CONFLICT (name) DO NOTHING;

ALTER TABLE books ADD COLUMN category_id BIGINT;
UPDATE books
SET category_id = categories.id
FROM categories
WHERE categories.name = CASE WHEN BTRIM(books.category) = '' THEN 'Uncategorized' ELSE BTRIM(books.category) END;
ALTER TABLE books ALTER COLUMN category_id SET NOT NULL;
ALTER TABLE books
    ADD CONSTRAINT fk_books_category FOREIGN KEY (category_id)
    REFERENCES categories (id) ON DELETE RESTRICT;
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

INSERT INTO book_authors (book_id, author_id)
SELECT books.id, authors.id
FROM books
CROSS JOIN LATERAL unnest(string_to_array(books.author, ',')) AS split(author_name)
JOIN authors ON authors.full_name = BTRIM(split.author_name)
ON CONFLICT DO NOTHING;

ALTER TABLE books DROP COLUMN author;
ALTER TABLE books DROP COLUMN category;
