package todos

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/mainawycliffe/todo-dockertest-golang-mongo-demo/model"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Client

const MONGO_INITDB_ROOT_USERNAME = "root"
const MONGO_INITDB_ROOT_PASSWORD = "password"

func TestMain(m *testing.M) {
	// Setup
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.BuildAndRunWithOptions("../Dockerfile", &dockertest.RunOptions{
		Name:         "todo-tests",
		PortBindings: map[docker.Port][]docker.PortBinding{"27017/tcp": {{"127.0.0.1", "27017"}}},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		var err error
		db, err = mongo.Connect(
			context.TODO(),
			options.Client().ApplyURI(
				fmt.Sprintf("mongodb://localhost:%s", resource.GetPort("27017/tcp")),
			),
		)
		if err != nil {
			return err
		}
		return db.Ping(context.TODO(), nil)
	})
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Run tests
	exitCode := m.Run()

	// Teardown
	// When you're done, kill and remove the container
	if err = pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// disconnect mongodb client
	if err = db.Disconnect(context.TODO()); err != nil {
		panic(err)
	}

	// Exit
	os.Exit(exitCode)
}

func TestAddTodo(t *testing.T) {
	todos := Todos{
		client: db,
	}
	createdAt := primitive.Timestamp{
		T: uint32(time.Now().Unix()),
	}
	todo := model.Todo{
		Todo:      "test",
		IsDone:    false,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	// add todo
	todo, err := todos.AddTodo(todo)
	// assert error is nil
	assert.Nil(t, err)
	// assert todo ID is not not nil
	assert.NotNil(t, todo.ID)
	// fetch todo from the database
	todoGet, err := todos.GetTodo(context.Background(), todo.ID.Hex())
	// assert error is nil
	assert.Nil(t, err)
	// assert todo is equal to the todo returned from the database
	assert.Equal(t, todoGet, todo)
}

func TestDeleteTodo(t *testing.T) {
	todos := Todos{
		client: db,
	}
	createdAt := primitive.Timestamp{
		T: uint32(time.Now().Unix()),
	}
	todo := model.Todo{
		Todo:      "Test Delete Todo",
		IsDone:    false,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	todoAdd, err := todos.AddTodo(todo)
	assert.Nil(t, err)

	var sess mongo.Session
	var ctx = context.Background()

	t.Run("delete todo", func(t *testing.T) {
		if sess, err = todos.client.StartSession(); err != nil {
			t.Fatal(err)
		}

		if err = sess.StartTransaction(); err != nil {
			t.Fatal(err)
		}

		if err = mongo.WithSession(ctx, sess, func(sc mongo.SessionContext) error {
			err = todos.DeleteTodo(sc, todoAdd.ID.Hex())
			assert.Nil(t, err)

			if err = sess.AbortTransaction(sc); err != nil {
				t.Fatal(err)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		sess.EndSession(ctx)
	})

	t.Run("update todo", func(t *testing.T) {
		if sess, err = todos.client.StartSession(); err != nil {
			t.Fatal(err)
		}

		if err = sess.StartTransaction(); err != nil {
			t.Fatal(err)
		}

		if err = mongo.WithSession(ctx, sess, func(sc mongo.SessionContext) error {
			todoGet, err := todos.GetTodo(sc, todoAdd.ID.Hex())
			assert.Nil(t, err)

			todoAdd.Todo = "do the dishes"
			err = todos.UpdateTodo(sc, todoAdd)
			assert.Nil(t, err)

			todoGet, err = todos.GetTodo(sc, todoAdd.ID.Hex())
			assert.Nil(t, err)
			assert.Equal(t, todoGet.Todo, todoAdd.Todo)

			return nil
		}); err != nil {
			t.Fatal(err)
		}
		sess.EndSession(ctx)
	})
}

func TestGetTodo(t *testing.T) {
	todos := Todos{
		client: db,
	}
	createdAt := primitive.Timestamp{
		T: uint32(time.Now().Unix()),
	}
	todo := model.Todo{
		Todo:      "Test Get Todo",
		IsDone:    false,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	todoAdd, err := todos.AddTodo(todo)
	assert.Nil(t, err)
	todoGet, err := todos.GetTodo(context.Background(), todoAdd.ID.Hex())
	assert.Nil(t, err)
	assert.Equal(t, todoGet.Todo, todo.Todo)
}

func TestGetTodos(t *testing.T) {
	todos := Todos{
		client: db,
	}
	createdAt := primitive.Timestamp{
		T: uint32(time.Now().Unix()),
	}
	todo := model.Todo{
		Todo:      "Test Get Todos",
		IsDone:    false,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	todoAdd, err := todos.AddTodo(todo)
	assert.Nil(t, err)
	assert.NotNil(t, todoAdd.ID)
	todoList, err := todos.GetTodos()
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, len(todoList), 1)
}

func TestToggleTodo(t *testing.T) {
	todos := Todos{
		client: db,
	}
	createdAt := primitive.Timestamp{
		T: uint32(time.Now().Unix()),
	}
	todo := model.Todo{
		Todo:      "Test Toggle Todo",
		IsDone:    false,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	todoAdd, err := todos.AddTodo(todo)
	assert.Nil(t, err)
	err = todos.ToggleTodo(context.Background(), todoAdd.ID.Hex())
	assert.Nil(t, err)
	todoGet, err := todos.GetTodo(context.Background(), todoAdd.ID.Hex())
	assert.Nil(t, err)
	assert.NotEqual(t, todoGet.IsDone, todo.IsDone)
}
