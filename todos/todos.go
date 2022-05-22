package todos

import (
	"context"

	"github.com/mainawycliffe/todo-dockertest-golang-mongo-demo/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Todos struct {
	client *mongo.Client
}

func (todos *Todos) AddTodo(todo model.Todo) (model.Todo, error) {
	collection := todos.client.Database("todos").Collection("todos")
	result, err := collection.InsertOne(context.Background(), todo)
	todo.ID = result.InsertedID.(primitive.ObjectID)
	return todo, err
}

func (todos *Todos) DeleteTodo(ctx context.Context, id string) error {
	collection := todos.client.Database("todos").Collection("todos")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = collection.DeleteOne(ctx, model.Todo{
		ID: objectID,
	})
	return err
}

func (todos *Todos) GetTodo(ctx context.Context, id string) (model.Todo, error) {
	todo := model.Todo{}
	collection := todos.client.Database("todos").Collection("todos")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return model.Todo{}, err
	}
	err = collection.FindOne(ctx, bson.M{
		"_id": objectID,
	}).Decode(&todo)
	return todo, err
}

func (todos *Todos) GetTodos() ([]model.Todo, error) {
	collection := todos.client.Database("todos").Collection("todos")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	var todoList []model.Todo
	if err := cursor.All(context.Background(), &todoList); err != nil {
		return nil, err
	}
	return todoList, nil
}

func (todos *Todos) ToggleTodo(ctx context.Context, id string) error {
	collection := todos.client.Database("todos").Collection("todos")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	todo, err := todos.GetTodo(ctx, id)
	if err != nil {
		return err
	}
	_, err = collection.UpdateOne(context.Background(), bson.M{
		"_id": objectID,
	}, bson.M{
		"$set": bson.M{
			"isDone": !todo.IsDone,
		},
	})
	return err
}

func (todos *Todos) UpdateTodo(ctx context.Context, todo model.Todo) error {
	collection := todos.client.Database("todos").Collection("todos")
	_, err := collection.UpdateOne(ctx, bson.M{
		"_id": todo.ID,
	}, bson.M{
		"$set": bson.M{
			"text": todo.Todo,
		},
	})
	return err
}
