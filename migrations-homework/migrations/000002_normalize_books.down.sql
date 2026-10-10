ALTER TABLE books ADD COLUMN author TEXT NOT NULL DEFAULT '';
ALTER TABLE books ADD COLUMN category TEXT NOT NULL DEFAULT '';

UPDATE books AS b
SET author = legacy.author_text,
    category = legacy.category_text
FROM book_legacy_values AS legacy
WHERE legacy.book_id = b.id;

-- Rows created after the up migration have no legacy snapshot; reconstruct
-- those values from the normalized relations where possible.
UPDATE books AS b
SET category = categories.name
FROM categories
WHERE categories.id = b.category_id
  AND NOT EXISTS (
      SELECT 1 FROM book_legacy_values AS legacy WHERE legacy.book_id = b.id
  );

UPDATE books AS b
SET author = COALESCE(authors.author_names, '')
FROM (
    SELECT book_authors.book_id, string_agg(authors.full_name, ', ' ORDER BY authors.id) AS author_names
    FROM book_authors
    JOIN authors ON authors.id = book_authors.author_id
    GROUP BY book_authors.book_id
) AS authors
WHERE authors.book_id = b.id
  AND NOT EXISTS (
      SELECT 1 FROM book_legacy_values AS legacy WHERE legacy.book_id = b.id
  );

DROP TABLE book_authors;
DROP TABLE book_legacy_values;
DROP INDEX IF EXISTS idx_books_category_id;
ALTER TABLE books DROP CONSTRAINT fk_books_category;
ALTER TABLE books DROP COLUMN category_id;
DROP TABLE authors;
DROP TABLE categories;
