INSERT INTO categories (name) VALUES
    ('Fiction'), ('Science'), ('Technology');

INSERT INTO authors (full_name, bio) VALUES
    ('Ada North', 'Writes thoughtful fiction.'),
    ('Mira Chen', 'Researcher and science writer.'),
    ('Jon Bell', 'Author of practical technology books.'),
    ('Sam Rivera', 'Writes across science and culture.'),
    ('Nora Patel', 'Novelist and essayist.');

INSERT INTO books (title, isbn, category_id, published_year, description)
SELECT seed.title, seed.isbn, categories.id, seed.published_year, seed.description
FROM (VALUES
    ('The Quiet Engine', '9780000000001', 'Technology', 2021, 'A guide to reliable systems.'),
    ('Maps of Tomorrow', '9780000000002', 'Science', 2020, 'An introduction to climate models.'),
    ('A Small Orbit', '9780000000003', 'Fiction', 2019, 'A story about a distant observatory.'),
    ('Signals and Stories', '9780000000004', 'Technology', 2022, 'How networks shape communication.'),
    ('The Living Archive', '9780000000005', 'Science', 2018, 'A field guide to biodiversity.'),
    ('Winter Letters', '9780000000006', 'Fiction', 2017, 'Letters from a changing city.'),
    ('Practical PostgreSQL', '9780000000007', 'Technology', 2023, 'Relational data in practice.'),
    ('Light Between Worlds', '9780000000008', 'Fiction', 2021, 'A novel of family and astronomy.'),
    ('Measured Earth', '9780000000009', 'Science', 2024, 'Observations from a warming planet.'),
    ('Designing Small Services', '9780000000010', 'Technology', 2022, 'A concise service design handbook.')
) AS seed(title, isbn, category_name, published_year, description)
JOIN categories ON categories.name = seed.category_name;

INSERT INTO book_authors (book_id, author_id)
SELECT books.id, authors.id
FROM (VALUES
    ('9780000000001', 'Jon Bell'),
    ('9780000000001', 'Ada North'),
    ('9780000000002', 'Mira Chen'),
    ('9780000000002', 'Sam Rivera'),
    ('9780000000003', 'Nora Patel'),
    ('9780000000004', 'Jon Bell'),
    ('9780000000005', 'Sam Rivera'),
    ('9780000000005', 'Mira Chen'),
    ('9780000000006', 'Nora Patel'),
    ('9780000000007', 'Jon Bell'),
    ('9780000000008', 'Ada North'),
    ('9780000000009', 'Mira Chen'),
    ('9780000000010', 'Jon Bell')
) AS links(isbn, author_name)
JOIN books ON books.isbn = links.isbn
JOIN authors ON authors.full_name = links.author_name;

INSERT INTO copies (book_id, inventory_no, status)
SELECT books.id, seed.inventory_no, seed.status
FROM (VALUES
    ('9780000000001', 'LIB-0001', 'available'),
    ('9780000000001', 'LIB-0002', 'checked_out'),
    ('9780000000002', 'LIB-0003', 'available'),
    ('9780000000002', 'LIB-0004', 'available'),
    ('9780000000003', 'LIB-0005', 'available'),
    ('9780000000003', 'LIB-0006', 'lost'),
    ('9780000000004', 'LIB-0007', 'available'),
    ('9780000000004', 'LIB-0008', 'checked_out'),
    ('9780000000005', 'LIB-0009', 'available'),
    ('9780000000005', 'LIB-0010', 'available'),
    ('9780000000006', 'LIB-0011', 'available'),
    ('9780000000006', 'LIB-0012', 'available'),
    ('9780000000007', 'LIB-0013', 'available'),
    ('9780000000007', 'LIB-0014', 'checked_out'),
    ('9780000000008', 'LIB-0015', 'available'),
    ('9780000000008', 'LIB-0016', 'available'),
    ('9780000000009', 'LIB-0017', 'available'),
    ('9780000000009', 'LIB-0018', 'available'),
    ('9780000000010', 'LIB-0019', 'available'),
    ('9780000000010', 'LIB-0020', 'available')
) AS seed(isbn, inventory_no, status)
JOIN books ON books.isbn = seed.isbn;

INSERT INTO users (email, role) VALUES ('librarian@libratrack.local', 'librarian');
