package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Demo struct {
	// File data
	Filename string
	URL      string
	Size     int64
	SizeStr  string
	// Match data
	Hostname    string
	Time        time.Time
	TimeStr     string
	Duration    int
	DurationStr string
	MapID       string
	MapName     string
	GameType    string
	IsDuel      bool
	Player1     string
	Player2     string
	Score1      int
	Score2      int
}

type ByDate []Demo

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool { return a[i].Time.After(a[j].Time) }

func datestr(date time.Time) string {
	return date.Format(time.RFC822)
}

func durationstr(duration int) string {
	return (time.Second * time.Duration(duration)).String()
}

func hrsize(size int64) string {
	prefixes := []string{"B", "kB", "MB", "GB", "TB"}
	fsize := float64(size)
	index := 0
	for fsize > 1024 && index < len(prefixes) {
		fsize /= 1024
		index++
	}
	return strconv.FormatFloat(fsize, 'f', 2, 64) + prefixes[index]
}

func getDemos() ([]Demo, error) {
	files, err := ioutil.ReadDir(*demos)
	if err != nil {
		return nil, err
	}

	var demolist []Demo
	for _, file := range files {
		// Get basic file data
		demo := Demo{
			Filename: file.Name(),
			URL:      "/demos/" + file.Name(),
			Size:     file.Size(),
			SizeStr:  hrsize(file.Size()),
		}
		// Read header and parse match metadata
		file, err := os.Open(filepath.Join(*demos, file.Name()))
		if err != nil {
			log.Printf("Could not open %s (??) Skipping file..\n", demo.Filename)
			continue
		}
		header := make([]byte, 1024)
		n, err := file.Read(header)
		if err != nil {
			log.Printf("Could not read from %s (??) Skipping file..\n", demo.Filename)
			continue
		}
		if n < 1024 {
			log.Printf("Managed to read only %f bytes from %s (??) Skipping file..\n", n, demo.Filename)
			continue
		}
		headerlen := int(header[0x30])
		headerdata := header[0x38 : 0x38+headerlen]
		for headerdata != nil {
			nextkey := bytes.IndexByte(headerdata, 0)
			if nextkey < 0 {
				break
			}
			keystr := string(headerdata[0:nextkey])

			headerdata = headerdata[nextkey+1:]
			nextvalue := bytes.IndexByte(headerdata, 0)
			if nextvalue < 0 {
				nextvalue = len(headerdata)
			}
			valuestr := string(headerdata[0:nextvalue])
			if nextvalue == len(headerdata) {
				headerdata = nil
			} else {
				headerdata = headerdata[nextvalue+1:]
			}

			switch keystr {
			case "hostname":
				demo.Hostname = valuestr
			case "localtime":
				unix, err := strconv.Atoi(valuestr)
				if err != nil {
					log.Printf("Invalid unixtime format '%s' in %s (??) Skipping field..\n", valuestr, demo.Filename)
					break
				}
				demo.Time = time.Unix(int64(unix), 0)
				demo.TimeStr = datestr(demo.Time)
			case "duration":
				duration, err := strconv.Atoi(valuestr)
				if err != nil {
					log.Printf("Invalid duration format '%s' in %s (??) Skipping field..\n", valuestr, demo.Filename)
					break
				}
				demo.Duration = duration
				demo.DurationStr = durationstr(duration)
			case "mapname":
				demo.MapID = valuestr
			case "levelname":
				demo.MapName = valuestr
			case "gametype":
				demo.GameType = valuestr
				demo.IsDuel = valuestr == "duel"
			case "matchscore":
				parts := strings.Split(valuestr, " : ")
				if len(parts) < 2 {
					log.Printf("Invalid match score format '%s' in %s (??) Skipping field..\n", valuestr, demo.Filename)
					break
				}
				score1, err := strconv.Atoi(parts[0])
				if err != nil {
					log.Printf("Invalid score1 format '%s' in %s (??) Skipping field..\n", parts[0], demo.Filename)
					break
				}
				score2, err := strconv.Atoi(parts[1])
				if err != nil {
					log.Printf("Invalid score format '%s' in %s (??) Skipping field..\n", parts[1], demo.Filename)
					break
				}
				demo.Score1 = score1
				demo.Score2 = score2
			case "matchname":
				if demo.IsDuel {
					parts := strings.Split(valuestr, " ^7vs ")
					if len(parts) < 2 {
						// Not a duel maybe?
						break
					}
					demo.Player1 = parts[0]
					demo.Player2 = parts[1]
				}
			}
		}
		demolist = append(demolist, demo)
	}

	return demolist, nil
}
