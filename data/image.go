package data

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/errors"
	"github.com/minio/minio-go"
	"github.com/pritunl/pritunl-cloud/database"
	"github.com/pritunl/pritunl-cloud/datacenter"
	"github.com/pritunl/pritunl-cloud/errortypes"
	"github.com/pritunl/pritunl-cloud/event"
	"github.com/pritunl/pritunl-cloud/image"
	"github.com/pritunl/pritunl-cloud/instance"
	"github.com/pritunl/pritunl-cloud/node"
	"github.com/pritunl/pritunl-cloud/storage"
	"github.com/pritunl/pritunl-cloud/utils"
	"github.com/pritunl/pritunl-cloud/vm"
	"github.com/pritunl/pritunl-cloud/zone"
	"gopkg.in/mgo.v2/bson"
	"path"
	"time"
)

func getImage(db *database.Database, img *image.Image,
	pth string) (err error) {

	store, err := storage.Get(db, img.Storage)
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"id":         img.Id.Hex(),
		"storage_id": store.Id.Hex(),
		"key":        img.Key,
		"path":       pth,
	}).Info("data: Downloading image")

	client, err := minio.New(
		store.Endpoint, store.AccessKey, store.SecretKey, !store.Insecure)
	if err != nil {
		err = &errortypes.ConnectionError{
			errors.Wrap(err, "data: Failed to connect to storage"),
		}
		return
	}

	err = client.FGetObject(store.Bucket,
		img.Key, pth, minio.GetObjectOptions{})
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "data: Failed to download image"),
		}
		return
	}

	return
}

func WriteImage(db *database.Database, imgId, dskId bson.ObjectId) (
	err error) {

	diskPath := vm.GetDiskPath(dskId)
	disksPath := vm.GetDisksPath()

	err = utils.ExistsMkdir(disksPath, 0755)
	if err != nil {
		return
	}

	img, err := image.Get(db, imgId)
	if err != nil {
		return
	}

	if img.Type == storage.Public {
		cacheDir := node.Self.GetCachePath()

		imagePth := path.Join(
			cacheDir,
			fmt.Sprintf("image-%s-%s", img.Id.Hex(), img.Etag),
		)

		err = utils.ExistsMkdir(cacheDir, 0755)
		if err != nil {
			return
		}

		exists, e := utils.Exists(imagePth)
		if e != nil {
			err = e
			return
		}

		if !exists {
			err = getImage(db, img, imagePth)
			if err != nil {
				return
			}
		}

		exists, err = utils.Exists(diskPath)
		if err != nil {
			return
		}

		if exists {
			logrus.WithFields(logrus.Fields{
				"image_id": img.Id.Hex(),
				"disk_id":  dskId.Hex(),
				"key":      img.Key,
				"path":     diskPath,
			}).Error("data: Blocking disk image overwrite")

			err = &errortypes.WriteError{
				errors.Wrap(err, "data: Image already exists"),
			}
			return
		}

		err = utils.Exec("", "cp", imagePth, diskPath)
		if err != nil {
			return
		}
	} else {
		exists, e := utils.Exists(diskPath)
		if e != nil {
			err = e
			return
		}

		if exists {
			logrus.WithFields(logrus.Fields{
				"image_id": img.Id.Hex(),
				"disk_id":  dskId.Hex(),
				"key":      img.Key,
				"path":     diskPath,
			}).Error("data: Blocking disk image overwrite")

			err = &errortypes.WriteError{
				errors.Wrap(err, "data: Image already exists"),
			}
			return
		}

		err = getImage(db, img, diskPath)
		if err != nil {
			return
		}
	}

	return
}

func DeleteImage(db *database.Database, imgId bson.ObjectId) (err error) {
	img, err := image.Get(db, imgId)
	if err != nil {
		return
	}

	if img.Type == storage.Public {
		return
	}

	store, err := storage.Get(db, img.Storage)
	if err != nil {
		return
	}

	client, err := minio.New(
		store.Endpoint, store.AccessKey, store.SecretKey, !store.Insecure)
	if err != nil {
		err = &errortypes.ConnectionError{
			errors.Wrap(err, "data: Failed to connect to storage"),
		}
		return
	}

	err = client.RemoveObject(store.Bucket, img.Key)
	if err != nil {
		return
	}

	err = image.Remove(db, img.Id)
	if err != nil {
		return
	}

	return
}

func DeleteImages(db *database.Database, imgIds []bson.ObjectId) (err error) {
	for _, imgId := range imgIds {
		err = DeleteImage(db, imgId)
		if err != nil {
			return
		}
	}

	return
}

func CreateSnapshot(db *database.Database,
	virt *vm.VirtualMachine) (err error) {

	disk := virt.Disks[0]
	cacheDir := node.Self.GetCachePath()

	logrus.WithFields(logrus.Fields{
		"instance_id": virt.Id.Hex(),
		"source_path": disk.Path,
	}).Info("data: Creating instance snapshot")

	inst, err := instance.Get(db, virt.Id)
	if err != nil {
		return
	}

	zne, err := zone.Get(db, inst.Zone)
	if err != nil {
		return
	}

	dc, err := datacenter.Get(db, zne.Datacenter)
	if err != nil {
		return
	}

	store, err := storage.Get(db, dc.PrivateStorage)
	if err != nil {
		return
	}

	imgId := bson.NewObjectId()
	tmpPath := path.Join(cacheDir,
		fmt.Sprintf("snapshot-%s", imgId.Hex()))
	img := &image.Image{
		Id: imgId,
		Name: fmt.Sprintf("%s-%s", inst.Name,
			time.Now().Format("2006-01-02T15:04:05")),
		Organization: inst.Organization,
		Type:         storage.Private,
		Storage:      store.Id,
		Key:          fmt.Sprintf("snapshot/%s.qcow2", imgId.Hex()),
	}

	defer utils.Remove(tmpPath)
	err = utils.Exec("", "qemu-img", "convert", "-f", "qcow2",
		"-O", "qcow2", "-c", disk.Path, tmpPath)
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"instance_id": virt.Id.Hex(),
		"source_path": disk.Path,
		"storage_id":  store.Id.Hex(),
		"object_key":  img.Key,
	}).Info("data: Uploading instance snapshot")

	client, err := minio.New(
		store.Endpoint, store.AccessKey, store.SecretKey, !store.Insecure)
	if err != nil {
		err = &errortypes.ConnectionError{
			errors.Wrap(err, "data: Failed to connect to storage"),
		}
		return
	}

	_, err = client.FPutObject(store.Bucket, img.Key, tmpPath,
		minio.PutObjectOptions{})
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "data: Failed to write object"),
		}
		return
	}

	obj, err := client.StatObject(store.Bucket, img.Key,
		minio.StatObjectOptions{})
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "data: Failed to stat object"),
		}
		return
	}

	img.Etag = image.GetEtag(obj)
	img.LastModified = obj.LastModified

	err = img.Insert(db)
	if err != nil {
		client.RemoveObject(store.Bucket, img.Key)
		return
	}

	event.PublishDispatch(db, "image.change")

	return
}
