package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DemoFile struct {
	Header map[string]string
	CSData map[int]string
}

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
		if strings.HasSuffix(file.Name(), ".wdz20") {
			demolist = append(demolist, getDemo(file))
		}
	}

	return demolist, nil
}

func getDemo(file os.FileInfo) Demo {
	filename := filepath.Join(*demos, file.Name())

	_, err := os.Stat(filename + ".dat")
	if err == nil {
		return readCachedDemo(filename + ".dat")
	}

	if !os.IsNotExist(err) {
		log.Printf("Error while calling stat() on %s: %s\n", filename+".dat", err.Error())
	}

	demofile := readDemoFile(filename)

	// Get basic file data
	demo := Demo{
		Filename: file.Name(),
		URL:      "/demos/" + file.Name(),
		Size:     file.Size(),
		SizeStr:  hrsize(file.Size()),
	}

	for key, value := range demofile.Header {
		switch key {
		case "hostname":
			demo.Hostname = value
		case "localtime":
			unix, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("Invalid unixtime format '%s' in %s (??) Skipping field..\n", value, demo.Filename)
				break
			}
			demo.Time = time.Unix(int64(unix), 0)
			demo.TimeStr = datestr(demo.Time)
		case "duration":
			duration, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("Invalid duration format '%s' in %s (??) Skipping field..\n", value, demo.Filename)
				break
			}
			demo.Duration = duration
			demo.DurationStr = durationstr(duration)
		case "mapname":
			demo.MapID = value
		case "levelname":
			demo.MapName = value
		case "gametype":
			demo.GameType = value
			demo.IsDuel = value == "duel"
		case "matchscore":
			parts := strings.Split(value, " : ")
			if len(parts) < 2 {
				log.Printf("Invalid match score format '%s' in %s (??) Skipping field..\n", value, demo.Filename)
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
				parts := strings.Split(value, " ^7vs ")
				if len(parts) < 2 {
					// Not a duel maybe?
					break
				}
				demo.Player1 = parts[0]
				demo.Player2 = parts[1]
			}
		}
	}
	if demo.Player1 == "" || demo.Player2 == "" {
		demo.Player1 = demofile.CSData[20]
		demo.Player2 = demofile.CSData[21]
	}

	saveCachedDemo(filename, demo)

	return demo
}

func saveCachedDemo(filename string, demo Demo) {
	file, err := os.Create(filename + ".dat")
	if err != nil {
		log.Printf("Could not create %s: %s\n", filename+".dat", err.Error())
		return
	}
	err = json.NewEncoder(file).Encode(demo)
	if err != nil {
		log.Printf("Could not serialize %s: %s\n", filename, err.Error())
		file.Close()
		os.Remove(filename + ".dat")
	}
}

func readCachedDemo(filename string) Demo {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		log.Printf("Could not open %s: %s\n", filename, err.Error())
	}
	var demo Demo
	json.NewDecoder(file).Decode(&demo)
	if err != nil {
		log.Printf("Could not deserialize %s: %s\n", filename, err.Error())
	}
	return demo
}

func readDemoFile(filename string) DemoFile {
	const gzipstart = 0x4043

	var out DemoFile
	out.Header = make(map[string]string)

	// Read header and parse match metadata
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		log.Printf("Could not open %s (??) Skipping file..\n", filename)
		return out
	}
	header := make([]byte, gzipstart)
	n, err := file.Read(header)
	if err != nil {
		log.Printf("Could not read from %s (??) Skipping file..\n", filename)
		return out
	}
	if n < gzipstart {
		log.Printf("Managed to read only %f bytes from %s (??) Skipping file..\n", n, filename)
		return out
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

		out.Header[keystr] = valuestr
	}

	reader, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("Could not read GZIP content from %s: %s\n", filename, err.Error())
		return out
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("Could not read all bytes from GZIP reader of %s: %s\n", filename, err.Error())
		return out
	}

	out.CSData = extractCS(data)

	return out
}

func extractCS(data []byte) map[int]string {
	out := make(map[int]string)
	// "..cs " (with first 2 bytes being non-ascii)
	cssep := []byte{0x00, 0x0B, 0x63, 0x73, 0x20}

	csdata := data[:]
	isend := false
	for {
		// Get next delimiter
		idx := bytes.Index(csdata, cssep)
		if idx < 0 {
			break
		}
		newidx := bytes.Index(csdata[idx+1:], cssep)
		if newidx < 0 {
			newidx = bytes.IndexByte(csdata[idx+1:], 0x00)
			isend = true
		}

		// Get current pair
		curdata := csdata[idx+5 : idx+1+newidx]

		// Split, separator is 0x20 (whitespace)
		curdatasep := bytes.IndexByte(curdata, 0x20)

		// Get pair
		curkey := curdata[:curdatasep]
		curval := curdata[curdatasep+1:]

		// Parse key as int
		intkey, err := strconv.Atoi(string(curkey))
		if err != nil {
			log.Printf("%s cannot be converted to int: %s\n", intkey, err.Error())
			continue
		}

		// Put pair in map (trimming the "" around the value)
		out[intkey] = strings.Trim(string(curval), "\"")

		if isend {
			break
		}
		csdata = csdata[idx+1:]
	}

	return out
}
