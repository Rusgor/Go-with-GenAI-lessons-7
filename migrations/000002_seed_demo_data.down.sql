DELETE FROM users WHERE email = 'librarian@libratrack.local';
DELETE FROM copies WHERE inventory_no BETWEEN 'LIB-0001' AND 'LIB-0020';
DELETE FROM books WHERE isbn BETWEEN '9780000000001' AND '9780000000010';
DELETE FROM authors WHERE full_name IN ('Ada North', 'Mira Chen', 'Jon Bell', 'Sam Rivera', 'Nora Patel');
DELETE FROM categories WHERE name IN ('Fiction', 'Science', 'Technology');
