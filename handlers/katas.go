package handlers

import (
	"io/ioutil"
	"log"
	"math/rand"
	"time"
	"vimkatas/models"
)

func SelectKata() (models.Kata, error) {
	katas, err := ioutil.ReadDir("./exercises/")
	if err != nil {
		log.Fatal(err)
	}

	var newKata models.Kata

	rand.Seed(time.Now().Unix())
	kata := katas[rand.Intn(len(katas))].Name()

	stdout, err := ioutil.ReadFile("./exercises/" + kata + "/out.js")
	tips, err := ioutil.ReadFile("./exercises/" + kata + "/tips.md")

	newKata.Kata = kata
	newKata.Tips = tips
	newKata.Example = stdout
	newKata.VimText = string("./exercises/" + kata + "/in.js")

	return newKata, nil
}