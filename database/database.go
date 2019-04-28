package database

import (
	"context"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/mongo"
	"github.com/pritunl/mongo-go-driver/mongo/options"
	"github.com/pritunl/pritunl-cloud/config"
	"github.com/pritunl/pritunl-cloud/constants"
	"github.com/pritunl/pritunl-cloud/errortypes"
	"github.com/pritunl/pritunl-cloud/requires"
)

var (
	Client          *mongo.Client
	DefaultDatabase string
)

type Database struct {
	ctx      context.Context
	client   *mongo.Client
	database *mongo.Database
}

func (d *Database) Deadline() (time.Time, bool) {
	if d.ctx != nil {
		return d.ctx.Deadline()
	}
	return time.Time{}, false
}

func (d *Database) Done() <-chan struct{} {
	if d.ctx != nil {
		return d.ctx.Done()
	}
	return nil
}

func (d *Database) Err() error {
	if d.ctx != nil {
		return d.ctx.Err()
	}
	return nil
}

func (d *Database) Value(key interface{}) interface{} {
	if d.ctx != nil {
		return d.ctx.Value(key)
	}
	return nil
}

func (d *Database) String() string {
	return "context.database"
}

func (d *Database) Close() {
}

func (d *Database) getCollection(name string) (coll *Collection) {
	coll = &Collection{
		db:         d,
		Collection: d.database.Collection(name),
	}
	return
}

func (d *Database) Users() (coll *Collection) {
	coll = d.getCollection("users")
	return
}

func (d *Database) Policies() (coll *Collection) {
	coll = d.getCollection("policies")
	return
}

func (d *Database) Devices() (coll *Collection) {
	coll = d.getCollection("devices")
	return
}

func (d *Database) Sessions() (coll *Collection) {
	coll = d.getCollection("sessions")
	return
}

func (d *Database) Tasks() (coll *Collection) {
	coll = d.getCollection("tasks")
	return
}

func (d *Database) Tokens() (coll *Collection) {
	coll = d.getCollection("tokens")
	return
}

func (d *Database) CsrfTokens() (coll *Collection) {
	coll = d.getCollection("csrf_tokens")
	return
}

func (d *Database) SecondaryTokens() (coll *Collection) {
	coll = d.getCollection("secondary_tokens")
	return
}

func (d *Database) Nonces() (coll *Collection) {
	coll = d.getCollection("nonces")
	return
}

func (d *Database) Settings() (coll *Collection) {
	coll = d.getCollection("settings")
	return
}

func (d *Database) Events() (coll *Collection) {
	coll = d.getCollection("events")
	return
}

func (d *Database) Nodes() (coll *Collection) {
	coll = d.getCollection("nodes")
	return
}

func (d *Database) Organizations() (coll *Collection) {
	coll = d.getCollection("organizations")
	return
}

func (d *Database) Storages() (coll *Collection) {
	coll = d.getCollection("storages")
	return
}

func (d *Database) Images() (coll *Collection) {
	coll = d.getCollection("images")
	return
}

func (d *Database) Datacenters() (coll *Collection) {
	coll = d.getCollection("datacenters")
	return
}

func (d *Database) Zones() (coll *Collection) {
	coll = d.getCollection("zones")
	return
}

func (d *Database) Instances() (coll *Collection) {
	coll = d.getCollection("instances")
	return
}

func (d *Database) Disks() (coll *Collection) {
	coll = d.getCollection("disks")
	return
}

func (d *Database) Blocks() (coll *Collection) {
	coll = d.getCollection("blocks")
	return
}

func (d *Database) BlocksIp() (coll *Collection) {
	coll = d.getCollection("blocks_ip")
	return
}

func (d *Database) Firewalls() (coll *Collection) {
	coll = d.getCollection("firewalls")
	return
}

func (d *Database) Vpcs() (coll *Collection) {
	coll = d.getCollection("vpcs")
	return
}

func (d *Database) VpcsIp() (coll *Collection) {
	coll = d.getCollection("vpcs_ip")
	return
}

func (d *Database) Authorities() (coll *Collection) {
	coll = d.getCollection("authorities")
	return
}

func (d *Database) Certificates() (coll *Collection) {
	coll = d.getCollection("certificates")
	return
}

func (d *Database) Domains() (coll *Collection) {
	coll = d.getCollection("domains")
	return
}

func (d *Database) DomainsRecord() (coll *Collection) {
	coll = d.getCollection("domains_record")
	return
}

func (d *Database) AcmeChallenges() (coll *Collection) {
	coll = d.getCollection("acme_challenges")
	return
}

func (d *Database) Logs() (coll *Collection) {
	coll = d.getCollection("logs")
	return
}

func (d *Database) Audits() (coll *Collection) {
	coll = d.getCollection("audits")
	return
}

func (d *Database) Geo() (coll *Collection) {
	coll = d.getCollection("geo")
	return
}

func Connect() (err error) {
	mongoUrl, err := url.Parse(config.Config.MongoUri)
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Failed to parse mongo uri"),
		}
		return
	}

	logrus.WithFields(logrus.Fields{
		"mongodb_host": mongoUrl.Host,
	}).Info("database: Connecting to MongoDB server")

	path := mongoUrl.Path
	if len(path) > 1 {
		DefaultDatabase = path[1:]
	}

	opts := options.Client().ApplyURI(config.Config.MongoUri)
	client, err := mongo.NewClient(opts)
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Client error"),
		}
		return
	}

	err = client.Connect(context.TODO())
	if err != nil {
		err = &ConnectionError{
			errors.Wrap(err, "database: Connection error"),
		}
		return
	}

	Client = client

	err = ValidateDatabase()
	if err != nil {
		Client = nil
		return
	}

	logrus.WithFields(logrus.Fields{
		"mongodb_host": mongoUrl.Host,
	}).Info("database: Connected to MongoDB server")

	return
}

func ValidateDatabase() (err error) {
	db := GetDatabase()

	cursor, err := db.database.ListCollections(
		db, &bson.M{})
	if err != nil {
		err = ParseError(err)
		return
	}
	defer cursor.Close(db)

	for cursor.Next(db) {
		item := &struct {
			Name string `bson:"name"`
		}{}
		err = cursor.Decode(item)
		if err != nil {
			err = ParseError(err)
			return
		}

		if item.Name == "servers" {
			err = &errortypes.DatabaseError{
				errors.New("database: Cannot connect to pritunl database"),
			}
			return
		}
	}

	err = cursor.Err()
	if err != nil {
		err = ParseError(err)
		return
	}

	return
}

func GetDatabase() (db *Database) {
	client := Client
	if client == nil {
		return
	}

	database := client.Database(DefaultDatabase)

	db = &Database{
		client:   client,
		database: database,
	}
	return
}

func GetDatabaseCtx(ctx context.Context) (db *Database) {
	client := Client
	if client == nil {
		return
	}

	database := client.Database(DefaultDatabase)

	db = &Database{
		ctx:      ctx,
		client:   client,
		database: database,
	}
	return
}

func addIndexes() (err error) {
	db := GetDatabase()
	defer db.Close()

	index := &Index{
		Collection: db.Users(),
		Keys: &bson.D{
			{"username", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Users(),
		Keys: &bson.D{
			{"type", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Users(),
		Keys: &bson.D{
			{"roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Users(),
		Keys: &bson.D{
			{"token", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Audits(),
		Keys: &bson.D{
			{"user", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Policies(),
		Keys: &bson.D{
			{"roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.CsrfTokens(),
		Keys: &bson.D{
			{"timestamp", 1},
		},
		Expire: 168 * time.Hour,
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.SecondaryTokens(),
		Keys: &bson.D{
			{"timestamp", 1},
		},
		Expire: 3 * time.Minute,
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Nodes(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Nonces(),
		Keys: &bson.D{
			{"timestamp", 1},
		},
		Expire: 24 * time.Hour,
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Devices(),
		Keys: &bson.D{
			{"user", 1},
			{"mode", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Devices(),
		Keys: &bson.D{
			{"provider", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Organizations(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Organizations(),
		Keys: &bson.D{
			{"roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Images(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Images(),
		Keys: &bson.D{
			{"organization", 1},
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Images(),
		Keys: &bson.D{
			{"storage", 1},
			{"key", 1},
		},
		Unique: true,
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Images(),
		Keys: &bson.D{
			{"disk", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Disks(),
		Keys: &bson.D{
			{"instance", 1},
			{"index", 1},
		},
		Unique: true,
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Disks(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Disks(),
		Keys: &bson.D{
			{"organization", 1},
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Disks(),
		Keys: &bson.D{
			{"node", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Domains(),
		Keys: &bson.D{
			{"domain", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.DomainsRecord(),
		Keys: &bson.D{
			{"domain", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.DomainsRecord(),
		Keys: &bson.D{
			{"node", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Datacenters(),
		Keys: &bson.D{
			{"organization", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Datacenters(),
		Keys: &bson.D{
			{"match_organizations", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.BlocksIp(),
		Keys: &bson.D{
			{"block", 1},
			{"ip", 1},
		},
		Unique: true,
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.BlocksIp(),
		Keys: &bson.D{
			{"instance", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Vpcs(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Vpcs(),
		Keys: &bson.D{
			{"organization", 1},
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Vpcs(),
		Keys: &bson.D{
			{"vpc_id", 1},
		},
		Unique: true,
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Vpcs(),
		Keys: &bson.D{
			{"datacenter", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.VpcsIp(),
		Keys: &bson.D{
			{"vpc", 1},
			{"ip", 1},
		},
		Unique: true,
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.VpcsIp(),
		Keys: &bson.D{
			{"vpc", 1},
			{"instance", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Sessions(),
		Keys: &bson.D{
			{"user", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Firewalls(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Firewalls(),
		Keys: &bson.D{
			{"organization", 1},
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Firewalls(),
		Keys: &bson.D{
			{"network_roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Firewalls(),
		Keys: &bson.D{
			{"organization", 1},
			{"network_roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Zones(),
		Keys: &bson.D{
			{"datacenter", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Authorities(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Authorities(),
		Keys: &bson.D{
			{"organization", 1},
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Authorities(),
		Keys: &bson.D{
			{"network_roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Authorities(),
		Keys: &bson.D{
			{"organization", 1},
			{"network_roles", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Instances(),
		Keys: &bson.D{
			{"node", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Instances(),
		Keys: &bson.D{
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Instances(),
		Keys: &bson.D{
			{"organization", 1},
			{"name", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}
	index = &Index{
		Collection: db.Instances(),
		Keys: &bson.D{
			{"vnc_display", 1},
		},
		Partial: &bson.M{
			"vnc_display": &bson.M{
				"$gt": 0,
			},
		},
		Unique: true,
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Tasks(),
		Keys: &bson.D{
			{"timestamp", 1},
		},
		Expire: 720 * time.Hour,
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Events(),
		Keys: &bson.D{
			{"channel", 1},
		},
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.AcmeChallenges(),
		Keys: &bson.D{
			{"timestamp", 1},
		},
		Expire: 3 * time.Minute,
	}
	err = index.Create()
	if err != nil {
		return
	}

	index = &Index{
		Collection: db.Geo(),
		Keys: &bson.D{
			{"t", 1},
		},
		Expire: 360 * time.Hour,
	}
	err = index.Create()
	if err != nil {
		return
	}

	return
}

func addCollections() (err error) {
	db := GetDatabase()
	defer db.Close()

	cursor, err := db.database.ListCollections(
		db, &bson.M{})
	if err != nil {
		err = ParseError(err)
		return
	}
	defer cursor.Close(db)

	for cursor.Next(db) {
		item := &struct {
			Name string `bson:"name"`
		}{}
		err = cursor.Decode(item)
		if err != nil {
			err = ParseError(err)
			return
		}

		if item.Name == "events" {
			return
		}
	}

	err = cursor.Err()
	if err != nil {
		err = ParseError(err)
		return
	}

	err = db.database.RunCommand(
		context.Background(),
		bson.D{
			{"create", "events"},
			{"capped", true},
			{"max", 1000},
			{"size", 5242880},
		},
	).Err()
	if err != nil {
		err = ParseError(err)
		return
	}

	return
}

func init() {
	module := requires.New("database")
	module.After("config")

	module.Handler = func() (err error) {
		for {
			e := Connect()
			if e != nil {
				logrus.WithFields(logrus.Fields{
					"error": e,
				}).Error("database: Connection error")
			} else {
				break
			}

			time.Sleep(constants.RetryDelay)
		}

		err = addCollections()
		if err != nil {
			return
		}

		err = addIndexes()
		if err != nil {
			return
		}

		return
	}
}
