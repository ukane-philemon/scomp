package mongodb

import (
	"errors"
	"fmt"
	"time"

	"github.com/ukane-philemon/scomp/graph/model"
	"github.com/ukane-philemon/scomp/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// CreateAdminAccount creates a new admin with the provided username and
// password. An ErrorInvalidRequest will be returned is the username already
// exists.
func (mdb *MongoDB) CreateAdminAccount(username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("%w: missing username or password", db.ErrorInvalidRequest)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("bcrypt.GenerateFromPassword error: %w", err)
	}

	adminInfo := &dbAdmin{
		Username:  username,
		Password:  string(passwordHash),
		CreatedAt: time.Now().Unix(),
	}

	_, err = mdb.adminCollection.InsertOne(mdb.ctx, adminInfo)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: please try another username", db.ErrorInvalidRequest)
		}
		return fmt.Errorf("adminCollection.InsertOne error: %w", err)
	}

	return nil
}

// Login checks that the provided username and password matches a record in the
// database and are correct. Returns db.ErrorInvalidRequest if the password or
// username does not match any record.
func (mdb *MongoDB) Login(username, password string) (*model.Admin, error) {
	var admin *dbAdmin
	err := mdb.adminCollection.FindOne(mdb.ctx, bson.M{usernameKey: username}).Decode(&admin)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: username or password is incorrect", db.ErrorInvalidRequest)
		}
		return nil, fmt.Errorf("adminCollection.FindOne error: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("%w: username or password is incorrect", db.ErrorInvalidRequest)
	}

	return &model.Admin{
		ID:       admin.ID.Hex(),
		Username: username,
	}, nil
}
