package pagination

import (
	"fmt"
	"time"
)

// UserModel implements the Item interface for example purposes
type UserModel struct {
	UserID    string
	Name      string
	Email     string
	CreatedAt time.Time
}

// Time implements Item.Time
func (u UserModel) Time() time.Time {
	return u.CreatedAt
}

// ID implements Item.ID
func (u UserModel) ID() string {
	return u.UserID
}

// Example demonstrates how to use cursor-based pagination
func Example() {
	// Create some sample users
	users := []UserModel{
		{
			UserID:    "user1",
			Name:      "John Doe",
			Email:     "john@example.com",
			CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:    "user2",
			Name:      "Jane Smith",
			Email:     "jane@example.com",
			CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:    "user3",
			Name:      "Bob Johnson",
			Email:     "bob@example.com",
			CreatedAt: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
		},
	}

	// Set up pagination parameters
	params := CursorParams{
		Limit: 2, // Only get 2 items per page
	}

	// Validate and set defaults
	params.Validate()

	// Create a pagination result for the first page (first 2 items)
	// Total count is 3 (all users)
	firstPage := NewResult(users[:2], 3)

	fmt.Println("First page items:", len(firstPage.Items))
	fmt.Println("Total count:", firstPage.TotalCount)
	fmt.Println("Has next cursor:", firstPage.NextCursor != nil)

	// In a real app, you would use the cursor to get the next page
	// For this example, we'll simulate getting the next page
	if firstPage.NextCursor != nil {
		// Use the cursor to create parameters for the next page
		nextParams := CursorParams{
			Cursor: firstPage.NextCursor,
			Limit:  2,
		}
		nextParams.Validate()

		// In a real app, you would query the database with these params
		// For this example, we'll just use the last item
		// Total count should be the same as before
		nextPage := NewResult(users[2:], 3)

		fmt.Println("\nNext page items:", len(nextPage.Items))
		fmt.Println("Total count:", nextPage.TotalCount)
		fmt.Println("Has next cursor:", nextPage.NextCursor != nil)
	}

	// Output:
	// First page items: 2
	// Total count: 3
	// Has next cursor: true
	//
	// Next page items: 1
	// Total count: 3
	// Has next cursor: false
}
