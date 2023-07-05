package dao

import (
	"fmt"
	"github.com/goodrain/rainbond/db/model"
	"github.com/jinzhu/gorm"
)

// AppGrayReleaseModeDaoImpl app gray release dao
type AppGrayReleaseModeDaoImpl struct {
	DB *gorm.DB
}

func (t *AppGrayReleaseModeDaoImpl) AddModel(mo model.Interface) error {
	gray, ok := mo.(*model.AppGrayRelease)
	if !ok {
		return fmt.Errorf("mo.(*model.AppGrayRelease) err")
	}
	return t.DB.Create(gray).Error
}

// UpdateModel update model
func (t *AppGrayReleaseModeDaoImpl) UpdateModel(mo model.Interface) error {
	gray, ok := mo.(*model.AppGrayRelease)
	if !ok {
		return fmt.Errorf("mo.(*model.AppGrayRelease) err")
	}
	return t.DB.Save(gray).Error
}

func (t *AppGrayReleaseModeDaoImpl) GetGrayRelease(appID string) (model.AppGrayRelease, error) {
	var gray model.AppGrayRelease
	if err := t.DB.Where("app_id=?", appID).First(&gray).Error; err != nil {
		return model.AppGrayRelease{}, err
	}
	return gray, nil
}
