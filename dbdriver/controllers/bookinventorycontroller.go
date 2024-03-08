package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/SakthiMahendran/SmartLibrary/dbdriver/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BookInventoryController struct {
	Client *mongo.Client
}

func NewBookInventoryController(client *mongo.Client) *BookInventoryController {
	return &BookInventoryController{Client: client}
}

func (bc *BookInventoryController) AddBook(bookID, bookName, author, bookDept string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inventoryCollection := bc.Client.Database("SMLS").Collection("BookInventory")
	booksCollection := bc.Client.Database("SMLS").Collection("Books")

	filter := bson.M{"book_id": bookID}
	var existingBook models.Book
	err := booksCollection.FindOne(ctx, filter).Decode(&existingBook)
	if err == nil {
		return fmt.Sprintf("Book with ID %s already exists", bookID), nil
	}

	filter = bson.M{"book_name": bookName}
	var existingInventory models.BookInventory
	err = inventoryCollection.FindOne(ctx, filter).Decode(&existingInventory)
	if err == nil {

		update := bson.M{"$inc": bson.M{"count": 1}}
		_, err := inventoryCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			return "", err
		}
	} else if err == mongo.ErrNoDocuments {

		newInventory := models.BookInventory{
			BookName:  bookName,
			Author:    author,
			BookDept:  bookDept,
			AddedDate: time.Now(),
			Count:     1,
		}
		_, err := inventoryCollection.InsertOne(ctx, newInventory)
		if err != nil {
			return "", err
		}
	} else {
		return "", err
	}

	newBook := models.Book{
		BookID:     bookID,
		BookStatus: true,
		BookInventoryPtr: &models.BookInventory{
			BookName:  bookName,
			Author:    author,
			BookDept:  bookDept,
			AddedDate: time.Now(),
			Count:     1,
		},
	}
	_, err = booksCollection.InsertOne(ctx, newBook)
	if err != nil {
		return "", err
	}

	_, err = booksCollection.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"book_inventory_ptr": newBook.BookInventoryPtr}})
	if err != nil {
		return "", err
	}

	return "Book added successfully", nil
}

func (bc *BookInventoryController) UpdateBook(presentData, toBeUpdatedData *models.BookInventory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inventoryCollection := bc.Client.Database("SMLS").Collection("BookInventory")
	booksCollection := bc.Client.Database("SMLS").Collection("Books")
	filter := bson.M{"book_name": presentData.BookName}
	var existingInventory models.BookInventory
	err := inventoryCollection.FindOne(ctx, filter).Decode(&existingInventory)
	if err != nil {
		return err
	}
	updateFields := bson.M{}
	if toBeUpdatedData.BookName != "" && toBeUpdatedData.BookName != existingInventory.BookName {
		updateFields["book_name"] = toBeUpdatedData.BookName
	}
	if toBeUpdatedData.Author != "" && toBeUpdatedData.Author != existingInventory.Author {
		updateFields["author"] = toBeUpdatedData.Author
	}
	if toBeUpdatedData.BookDept != "" && toBeUpdatedData.BookDept != existingInventory.BookDept {
		updateFields["book_dept"] = toBeUpdatedData.BookDept
	}
	if !toBeUpdatedData.AddedDate.IsZero() {
		updateFields["added_date"] = toBeUpdatedData.AddedDate
	}
	if toBeUpdatedData.Count != existingInventory.Count {
		updateFields["count"] = toBeUpdatedData.Count
	}
	update := bson.M{"$set": updateFields}
	_, err = inventoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	var updatedInventory models.BookInventory
	err = inventoryCollection.FindOne(ctx, filter).Decode(&updatedInventory)
	if err != nil {
		return err
	}

	filter = bson.M{"book_inventory_ptr.book_name": existingInventory.BookName}
	_, err = booksCollection.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"book_inventory_ptr": updatedInventory}})
	if err != nil {
		return err
	}

	return nil
}

func (bc *BookInventoryController) DeleteBook(bookID string) error {
	// Delete book and update inventory
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inventoryCollection := bc.Client.Database("SMLS").Collection("BookInventory")
	booksCollection := bc.Client.Database("SMLS").Collection("Books")

	// Search for book by book_id in the books collection
	filter := bson.M{"book_id": bookID}
	var book models.Book
	err := booksCollection.FindOne(ctx, filter).Decode(&book)
	if err != nil {
		// Book not found, return error
		fmt.Println("Book not present")
		return err
	}

	// Fetch the book name from the book inventory pointer
	bookName := book.BookInventoryPtr.BookName

	// Delete book from books collection
	_, err = booksCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	// Search for the book in the book inventory by book name and update count
	filter = bson.M{"book_name": bookName}
	update := bson.M{"$inc": bson.M{"count": -1}}
	_, err = inventoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// Fetch the updated book inventory data
	var updatedInventory models.BookInventory
	err = inventoryCollection.FindOne(ctx, filter).Decode(&updatedInventory)
	if err != nil {
		return err
	}

	// Reflect changes in the Books collection
	// Update the corresponding documents in the Books collection
	filter = bson.M{"book_inventory_ptr.book_name": bookName}
	_, err = booksCollection.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"book_inventory_ptr": updatedInventory}})
	if err != nil {
		return err
	}

	return nil
}

func (bc *BookInventoryController) GetBookCount(bookName string) (int, error) {
	// Get count of books by name
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inventoryCollection := bc.Client.Database("library").Collection("book_inventory")

	filter := bson.M{"book_name": bookName}
	count, err := inventoryCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func (bc *BookInventoryController) GetCategoryCount(category string) (int, error) {
	// Get count of books by category
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inventoryCollection := bc.Client.Database("library").Collection("book_inventory")

	filter := bson.M{"book_dept": category}
	count, err := inventoryCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func (bc *BookInventoryController) FindCategory(category string) ([]models.BookInventory, error) {
	// Find books by category
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inventoryCollection := bc.Client.Database("library").Collection("book_inventory")

	filter := bson.M{"book_dept": category}
	cursor, err := inventoryCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var books []models.BookInventory
	if err := cursor.All(ctx, &books); err != nil {
		return nil, err
	}

	return books, nil
}

func (bc *BookInventoryController) IsAvailable(bookName string) (bool, error) {
	// Check if book is available
	count, err := bc.GetBookCount(bookName)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (bc *BookInventoryController) Borrow(bookID primitive.ObjectID, student *models.Student) error {
	// Borrow book
	// Here you would update the book status and link it with the student who borrowed it
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	booksCollection := bc.Client.Database("library").Collection("books")

	// Update book status in books collection
	filter := bson.M{"_id": bookID}
	update := bson.M{"$set": bson.M{"book_status": false}}
	_, err := booksCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (bc *BookInventoryController) Return(bookID primitive.ObjectID, student *models.Student) error {
	// Return book
	// Here you would update the book status and remove the link with the student
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	booksCollection := bc.Client.Database("library").Collection("books")

	// Update book status in books collection
	filter := bson.M{"_id": bookID}
	update := bson.M{"$set": bson.M{"book_status": true}}
	_, err := booksCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}
