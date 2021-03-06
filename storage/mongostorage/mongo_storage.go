package mongostorage

import (
	"context"
	"errors"
	"fmt"
	"program/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	client          *mongo.Client
	collectionJokes *mongo.Collection
	collectionUsers *mongo.Collection
}

func NewMongoStorage(connectURI string) (*MongoStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// credential := options.Credential{
	// 	Username: os.Getenv("MONGO_INITDB_ROOT_USERNAME"),
	// 	Password: os.Getenv("MONGO_INITDB_ROOT_PASSWORD"),
	// }
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectURI))
	if err != nil {
		return nil, fmt.Errorf(" error while connecting to mongo: %v", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("pinging mongo: %w", err)
	}

	db := client.Database("mongoData")
	_ = db.CreateCollection(ctx, "Jokes")
	_ = db.CreateCollection(ctx, "Users")

	ms := &MongoStorage{
		client:          client,
		collectionJokes: db.Collection("Jokes"),
		collectionUsers: db.Collection("Users"),
	}

	model := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "title", Value: "text"},
				{Key: "body", Value: "text"},
			}},
		{
			Keys: bson.D{
				{Key: "score", Value: -1}},
		},
	}
	_, err = ms.collectionJokes.Indexes().CreateMany(context.TODO(), model)
	if err != nil {

		return nil, err
	}

	return ms, nil
}
func (ms *MongoStorage) FindID(id string) (model.Joke, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var j model.Joke

	err := ms.collectionJokes.FindOne(ctx, bson.M{"id": id}).Decode(&j)
	if err != nil {

		if err == mongo.ErrNoDocuments {

			return model.Joke{}, mongo.ErrNoDocuments
		}
		return model.Joke{}, fmt.Errorf("failed to execute query,error:%w", err)
	}

	return j, nil

}

func (ms *MongoStorage) Funniest(limit int) ([]model.Joke, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var j []model.Joke

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "score", Value: -1}})
	opts.SetLimit(int64(limit))
	result, err := ms.collectionJokes.Find(ctx, bson.D{}, opts)
	if err != nil {

		return nil, err
	}

	if err = result.All(ctx, &j); err != nil {

		return nil, err
	}

	return j, nil
}

func (ms *MongoStorage) Random(limit int) ([]model.Joke, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var j []model.Joke

	result, err := ms.collectionJokes.Aggregate(context.Background(), []bson.M{{"$sample": bson.M{"size": limit}}})
	if err != nil {
		return nil, nil
	}

	if err = result.All(ctx, &j); err != nil {
		return nil, err
	}

	return j, nil
}

func (ms *MongoStorage) TextSearch(text string) ([]model.Joke, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var j []model.Joke

	// filter := bson.D{		//search without indexes
	// 	{"$or", bson.A{
	// 		bson.D{{"body", primitive.Regex{Pattern: text, Options: "i"}}},
	// 		bson.D{{"title", primitive.Regex{Pattern: text, Options: "i"}}},
	// 	}},
	// }

	filter := bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: text}}}} //for indexModel

	result, err := ms.collectionJokes.Find(ctx, filter)
	if err != nil {

		return nil, err
	}

	if err = result.All(ctx, &j); err != nil {

		return nil, err
	} else if len(j) == 0 {

		return nil, mongo.ErrNoDocuments
	}

	return j, nil

}

func (ms *MongoStorage) AddJoke(j model.Joke) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := ms.collectionJokes.InsertOne(ctx, j)
	if err != nil {

		return err
	}
	return nil
}

func (ms *MongoStorage) UpdateByID(text string, id string) error {

	opts := options.Update().SetUpsert(false)
	filter := bson.D{{Key: "id", Value: id}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "body", Value: text}}}}

	_, err := ms.collectionJokes.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {

		return err
	}

	return nil
}

func (ms *MongoStorage) CloseClientDB() error {

	err := ms.client.Disconnect(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func (ms *MongoStorage) IsExists(user model.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	count, err := ms.collectionUsers.CountDocuments(ctx, bson.M{"username": user.Username})
	defer cancel()
	if err != nil {

		return err
	}

	if count > 0 {

		return errors.New("this username already exists")
	}
	return nil
}

func (ms *MongoStorage) CreateUser(user model.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, insertErr := ms.collectionUsers.InsertOne(ctx, user)
	defer cancel()
	if insertErr != nil {

		return insertErr
	}
	return nil
}

func (ms *MongoStorage) LoginUser(user model.User) (model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var foundUser model.User
	err := ms.collectionUsers.FindOne(ctx, bson.M{"username": user.Username}).Decode(&foundUser)
	defer cancel()
	if err != nil {

		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.User{}, mongo.ErrNoDocuments
		}
		return model.User{}, fmt.Errorf("failed to execute query,error:%w", err)
	}
	return foundUser, nil
}
func (ms *MongoStorage) UpdateTokens(signedToken string, signedRefreshToken string, username string) error {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	var updateObj primitive.D

	updateObj = append(updateObj, bson.E{Key: "token", Value: signedToken})
	updateObj = append(updateObj, bson.E{Key: "refresh_token", Value: signedRefreshToken})

	Updated_at, err := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}
	updateObj = append(updateObj, bson.E{Key: "updated_at", Value: Updated_at})

	upsert := true
	filter := bson.M{"username": username}
	opt := options.UpdateOptions{
		Upsert: &upsert,
	}

	_, err = ms.collectionUsers.UpdateOne(
		ctx,
		filter,
		bson.D{
			{Key: "$set", Value: updateObj},
		},
		&opt,
	)

	if err != nil {

		return err
	}

	return nil
}
func (ms *MongoStorage) MonthAndCount(year, count int) (int, int, error) {
	return -1, -1, fmt.Errorf("not implemented")
}
func (ms *MongoStorage) JokesByMonth(monthNumber int) (int, error) {
	return -1, fmt.Errorf("not implemented")
}
func (ms *MongoStorage) UsersWithoutJokes() ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
