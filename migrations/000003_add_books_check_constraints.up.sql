ALTER TABLE
    books
ADD
    CONSTRAINT books_isbn_check CHECK (isbn >= 0);

ALTER TABLE
    books
ADD
    CONSTRAINT books_year_check CHECK (
        year BETWEEN 1888
        AND date_part('year', now())
    );

ALTER TABLE
    books
ADD
    CONSTRAINT genres_length_check CHECK (
        array_length(genres, 1) BETWEEN 1
        AND 5
    );