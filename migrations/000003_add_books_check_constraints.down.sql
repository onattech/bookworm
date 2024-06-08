ALTER TABLE
    books DROP CONSTRAINT IF EXISTS books_isbn_check;

ALTER TABLE
    books DROP CONSTRAINT IF EXISTS books_year_check;

ALTER TABLE
    books DROP CONSTRAINT IF EXISTS genres_length_check;