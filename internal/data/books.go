package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"bookworm.onatim.com/internal/validator"
	"github.com/lib/pq"
)

type Book struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	// Use the ISBN type instead of int32. Note that the omitempty directive will
	// still work on this: if the ISBN field has the underlying value 0, then it will
	// be considered empty and omitted -- and the MarshalJSON() method we just made
	// won't be called at all.
	ISBN    ISBN     `json:"ISBN,omitempty"`
	Genres  []string `json:"genres,omitempty"`
	Version int32    `json:"version"`
}

func ValidateBook(v *validator.Validator, book *Book) {
	v.Check(book.Title != "", "title", "must be provided")
	v.Check(len(book.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(book.Year != 0, "year", "must be provided")
	v.Check(book.Year >= 1888, "year", "must be greater than 1888")
	v.Check(book.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(book.ISBN != 0, "ISBN", "must be provided")
	v.Check(book.ISBN > 0, "ISBN", "must be a positive integer")

	v.Check(book.Genres != nil, "genres", "must be provided")
	v.Check(len(book.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(book.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(book.Genres), "genres", "must not contain duplicate values")
}

// Define a BookModel struct type which wraps a sql.DB connection pool.
type BookModel struct {
	DB *sql.DB
}

// The Insert() method accepts a pointer to a book struct, which should contain the
// data for the new record.
func (m BookModel) Insert(book *Book) error {
	// Define the SQL query for inserting a new record in the books table and returning
	// the system-generated data.
	query := `
		INSERT INTO books (title, year, isbn, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`

	// Create an args slice containing the values for the placeholder parameters from
	// the book struct. Declaring this slice immediately next to our SQL query helps to
	// make it nice and clear *what values are being used where* in the query.
	args := []any{book.Title, book.Year, book.ISBN, pq.Array(book.Genres)}

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use the QueryRowContext() method and pass the context as the first argument to
	// execute the SQL query on our connection pool, passing in the args slice as a
	// variadic parameter and scanning the system-generated id, created_at and version
	// values into the book struct.
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&book.ID, &book.CreatedAt, &book.Version)
}

func (m BookModel) Get(id int64) (*Book, error) {
	// The PostgreSQL bigserial type that we're using for the book ID starts
	// auto-incrementing at 1 by default, so we know that no books will have ID values
	// less than that. To avoid making an unnecessary database call, we take a shortcut
	// and return an ErrRecordNotFound error straight away.
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	// Define the SQL query for retrieving the book data.
	query := `
		SELECT id, created_at, title, year, isbn, genres, version
		FROM books
		WHERE id = $1`

	// Declare a Book struct to hold the data returned by the query.
	var book Book

	// Use the context.WithTimeout() function to create a context.Context which carries a
	// 3-second timeout deadline. Note that we're using the empty context.Background()
	// as the 'parent' context.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query using the QueryRow() method, passing in the provided id value
	// as a placeholder parameter, and scan the response data into the fields of the
	// Book struct. Importantly, notice that we need to convert the scan target for the
	// genres column using the pq.Array() adapter function again.
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
		&book.CreatedAt,
		&book.Title,
		&book.Year,
		&book.ISBN,
		pq.Array(&book.Genres),
		&book.Version,
	)

	// Handle any errors. If there was no matching book found, Scan() will return
	// a sql.ErrNoRows error. We check for this and return our custom ErrRecordNotFound
	// error instead.
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// Otherwise, return a pointer to the Book struct.
	return &book, nil
}

func (m BookModel) Update(book *Book) error {
	// Declare the SQL query for updating the record and returning the new version
	// number.
	query := `
		UPDATE books
		SET title = $1, year = $2, isbn = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version`

	// Create an args slice containing the values for the placeholder parameters.
	args := []any{
		book.Title,
		book.Year,
		book.ISBN,
		pq.Array(book.Genres),
		book.ID,
		book.Version,
	}

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use QueryRowContext() and pass the context as the first argument. If
	// no matching row could be found, we know the book version has changed
	// (or the record has been deleted) and we return our custom ErrEditConflict error.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&book.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m BookModel) Delete(id int64) error {
	// Return an ErrRecordNotFound error if the book ID is less than 1.
	if id < 1 {
		return ErrRecordNotFound
	}

	// Construct the SQL query to delete the record.
	query := `
		DELETE FROM books
		WHERE id = $1`

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the SQL query using the ExecContext() method and pass the context
	// as the first argument. Passing in the id variable as the value for the
	// placeholder parameter. The Exec() method returns a sql.Result object.
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	// Call the RowsAffected() method on the sql.Result object to get the number of rows
	// affected by the query.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// If no rows were affected, we know that the books table didn't contain a record
	// with the provided ID at the moment we tried to delete it. In that case we
	// return an ErrRecordNotFound error.
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// GetAll() method returns a slice of books.
func (m BookModel) GetAll(title string, genres []string, filters Filters) ([]*Book, Metadata, error) {
	// Construct the SQL query to retrieve all book records.
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, title, year, isbn, genres, version
		FROM books
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Slice for placeholder parameters for the SQL query
	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}

	// Use QueryContext() to execute the query. This returns a sql.Rows resultset
	// containing the result.
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}

	// Importantly, defer a call to rows.Close() to ensure that the resultset is closed
	// before GetAll() returns.
	defer rows.Close()

	// Declare a totalRecords variable.
	totalRecords := 0

	// Initialize an empty slice to hold the book data.
	books := []*Book{}

	// Use rows.Next to iterate through the rows in the resultset.
	for rows.Next() {
		// Initialize an empty Book struct to hold the data for an individual book.
		var book Book

		// Scan the values from the row into the Book struct. Again, note that we're
		// using the pq.Array() adapter on the genres field here.
		err := rows.Scan(
			&totalRecords, // Scan the count from the window function into totalRecords.
			&book.ID,
			&book.CreatedAt,
			&book.Title,
			&book.Year,
			&book.ISBN,
			pq.Array(&book.Genres),
			&book.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		// Add the Book struct to the slice.
		books = append(books, &book)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// Generate a Metadata struct, passing in the total record count and pagination
	// parameters from the client.
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	// If everything went OK, then return the slice of books along with metadata.
	return books, metadata, nil
}
