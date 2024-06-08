package main

import (
	"errors"
	"fmt"
	"net/http"

	"bookworm.onatim.com/internal/data"
	"bookworm.onatim.com/internal/validator"
)

// Add a createBookHandler for the "POST /v1/books" endpoint.
func (app *application) createBookHandler(w http.ResponseWriter, r *http.Request) {
	// Declare an anonymous struct to hold the information that we expect to be in the
	// HTTP request body (note that the field names and types in the struct are a subset
	// of the Book struct that we created earlier). This struct will be our *target
	// decode destination*.
	var input struct {
		Title  string    `json:"title"`
		Year   int32     `json:"year"`
		ISBN   data.ISBN `json:"ISBN"`
		Genres []string  `json:"genres"`
	}

	// Use the readJSON() helper to decode the request body into the input struct.
	// If this returns an error we send the client the error message along with a 400
	// Bad Request status code, just like before.
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Copy the values from the input struct to a new Book struct.
	book := &data.Book{
		Title:  input.Title,
		Year:   input.Year,
		ISBN:   input.ISBN,
		Genres: input.Genres,
	}

	// Initialize a new Validator instance
	v := validator.New()

	// Call the ValidateBook() function and return a response containing the errors if
	// any of the checks fail.
	if data.ValidateBook(v, book); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Call the Insert() method on our books model, passing in a pointer to the
	// validated book struct. This will create a record in the database and update the
	// book struct with the system-generated information.
	err = app.models.Books.Insert(book)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// When sending a HTTP response, we want to include a Location header to let the
	// client know which URL they can find the newly-created resource at. We make an
	// empty http.Header map and then use the Set() method to add a new Location header,
	// interpolating the system-generated ID for our new book in the URL.
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/books/%d", book.ID))

	// Write a JSON response with a 201 Created status code, the book data in the
	// response body, and the Location header.
	err = app.writeJSON(w, http.StatusCreated, envelope{"book": book}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Add a showBookHandler for the "GET /v1/books/:id" endpoint.
func (app *application) showBookHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the book ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Call the Get() method to fetch the data for a specific book. We also need to
	// use the errors.Is() function to check if it returns a data.ErrRecordNotFound
	// error, in which case we send a 404 Not Found response to the client.
	book, err := app.models.Books.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Encode the struct to JSON and send it as the HTTP response.
	err = app.writeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateBookHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the book ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch the existing book record from the database, sending a 404 Not Found
	// response to the client if we couldn't find a matching record.
	book, err := app.models.Books.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Declare an input struct to hold the expected data from the client.
	var input struct {
		Title  *string    `json:"title"`
		Year   *int32     `json:"year"`
		ISBN   *data.ISBN `json:"ISBN"`
		Genres []string   `json:"genres"`
	}

	// Read the JSON request body data into the input struct.
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// If the input.Title value is nil then we know that no corresponding "title" key/
	// value pair was provided in the JSON request body. So we move on and leave the
	// book record unchanged. Otherwise, we update the book record with the new title
	// value. Importantly, because input.Title is a now a pointer to a string, we need
	// to dereference the pointer using the * operator to get the underlying value
	// before assigning it to our book record.
	if input.Title != nil {
		book.Title = *input.Title
	}

	// We also do the same for the other fields in the input struct.
	if input.Year != nil {
		book.Year = *input.Year
	}
	if input.ISBN != nil {
		book.ISBN = *input.ISBN
	}
	if input.Genres != nil {
		book.Genres = input.Genres // Note that we don't need to dereference a slice.
	}

	// Validate the updated book record, sending the client a 422 Unprocessable Entity
	// response if any checks fail.
	v := validator.New()

	if data.ValidateBook(v, book); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Pass the updated book record to our Update() method.
	err = app.models.Books.Update(book)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Write the updated book record in a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the book ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Delete the book from the database, sending a 404 Not Found response to the
	// client if there isn't a matching record.
	err = app.models.Books.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return a 200 OK status code along with a success message.
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "book successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listBooksHandler(w http.ResponseWriter, r *http.Request) {
	// To keep things consistent with our other handlers, we'll define an input struct
	// to hold the expected values from the request query string.
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	// Initialize a new Validator instance.
	v := validator.New()

	// Call r.URL.Query() to get the url.Values map containing the query string data.
	qs := r.URL.Query()

	// Use our helpers to extract the title and genres query string values, falling back
	// to defaults of an empty string and an empty slice respectively if they are not
	// provided by the client.
	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})

	// Get the page and page_size query string values as integers. Notice that we set
	// the default page value to 1 and default page_size to 20, and that we pass the
	// validator instance as the final argument here.
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	// Extract the sort query string value, falling back to "id" if it is not provided
	// by the client (which will imply a ascending sort on book ID).
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year", "isbn", "-id", "-title", "-year", "-isbn"}

	// Check the Validator instance for any errors and use the failedValidationResponse()
	// helper to send the client a response if necessary.
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Accept the metadata struct as a return value.
	books, metadata, err := app.models.Books.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Include the metadata in the response envelope.
	err = app.writeJSON(w, http.StatusOK, envelope{"books": books, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
