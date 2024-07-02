package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ukane-philemon/scomp/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const usernameKey = "username"

type Admin struct {
	ID             string `json:"_id" bson:"_id"`
	Username       string `json:"username" bson:"username"`
	HashedPassword string `json:"hashedPassword" bson:"hashedPassword"`
	CreatedAt      int64  `json:"createdAt" bson:"createdAt"`
}

// AdminRepository implements Repository.
type AdminRepository struct {
	ctx             context.Context
	adminCollection *mongo.Collection
}

// NewRepository creates a new instance of *AdminRepo.
func NewRepository(ctx context.Context, db *mongo.Database) (Repository, error) {
	adminCollectionIndex := mongo.IndexModel{
		Keys: bson.D{{
			Key:   usernameKey,
			Value: 1,
		}},
		Options: options.Index().SetUnique(true),
	}

	// Create a unique index on the admin collection.
	adminCollection := db.Collection("admin")
	_, err := adminCollection.Indexes().CreateOne(ctx, adminCollectionIndex)
	if err != nil {
		return nil, err
	}

	return &AdminRepository{
		ctx:             ctx,
		adminCollection: adminCollection,
	}, nil
}

// CreateAccount implements Repository.
func (ar *AdminRepository) CreateAccount(username, password string) (string, error) {
	if username == "" || password == "" {
		return "", fmt.Errorf("%w: missing username or password", db.ErrorInvalidRequest)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt.GenerateFromPassword error: %w", err)
	}

	adminInfo := &Admin{
		ID:             primitive.NewObjectID().Hex(),
		Username:       username,
		HashedPassword: string(passwordHash),
		CreatedAt:      time.Now().Unix(),
	}

	res, err := ar.adminCollection.InsertOne(ar.ctx, adminInfo)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", fmt.Errorf("%w: please try another username", db.ErrorInvalidRequest)
		}
		return "", fmt.Errorf("adminCollection.InsertOne error: %w", err)
	}

	return res.InsertedID.(string), nil
}

// LoginAccount implements Repository.
func (a *AdminRepository) LoginAccount(username, password string) (string, error) {
	var admin *Admin
	err := a.adminCollection.FindOne(a.ctx, bson.M{usernameKey: username}).Decode(&admin)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", fmt.Errorf("%w: username or password is incorrect", db.ErrorInvalidRequest)
		}
		return "", fmt.Errorf("adminCollection.FindOne error: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(admin.HashedPassword), []byte(password))
	if err != nil {
		return "", fmt.Errorf("%w: username or password is incorrect", db.ErrorInvalidRequest)
	}

	return admin.ID, nil
}
