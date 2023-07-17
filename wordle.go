package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	wordLen = 5
)

func chars() []byte {
	return []byte{
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
		'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'}
}

type wordSet map[string]bool

// key: letter, value: map of words
type byteMap map[byte]wordSet

// index: position in word
type positionMap []byteMap

func populateMaps(words wordSet) (positionMap, byteMap) {
	chars := chars()

	posMap := make(positionMap, wordLen)
	for idx := range posMap {
		posMap[idx] = make(byteMap)
		for _, c := range chars {
			posMap[idx][c] = make(wordSet)
		}
	}

	anyMap := make(byteMap)
	for _, c := range chars {
		anyMap[c] = make(wordSet)
	}

	for word := range words {
		for idx, b := range word {
			posMap[idx][byte(b)][word] = true
			anyMap[byte(b)][word] = true
		}
	}

	return posMap, anyMap
}

func readWords(path string) (wordSet, error) {
	words := make(wordSet)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words[scanner.Text()] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return words, nil
}

type choices struct {
	knownInPosition    map[int]byte
	knownOutOfPosition []byte
	knownBad           []byte
}

type solver struct {
	*choices
	posMap positionMap
	anyMap byteMap
	words  wordSet
}

// Mutates the smaller of s1 and s2 and returns that set.
func intersect(s1 wordSet, s2 wordSet) wordSet {
	smaller := s1
	bigger := s2

	if len(s1) > len(s2) {
		smaller = s2
		bigger = s1
	}

	for word := range smaller {
		if _, ok := bigger[word]; !ok {
			delete(smaller, word)
		}
	}

	return smaller
}

// Note that this approach can still suggest impossible words. The position
// where the known out of position letters were discovered is not taken into
// account. In other words, if a letter is 'yellow' in position 0, we'd want
// to only consider words where that letter is found in positions 1-4. The
// current approach, using the any map, will still consider position 0.
// The various maps in this data structure are mutated and are no longer
// truthful to input data.
func (s *solver) suggestWords() []string {
	// Start with every word as a candidate.
	// `words` may get mutated.
	candidates := s.words

	// Consider the words matching the known positions, and intersect those.
	for i := 0; i < wordLen; i++ {
		if char, ok := s.knownInPosition[i]; ok {
			candidates = intersect(candidates, s.posMap[i][char])
		}
	}

	// Consider the words matching out of positions, and intersect those.
	for _, char := range s.knownOutOfPosition {
		candidates = intersect(candidates, s.anyMap[char])
	}

	// Filter out impossible words.
	for _, char := range s.knownBad {
		for word := range s.anyMap[char] {
			delete(candidates, word)
		}
	}

	var res []string
	for word := range candidates {
		res = append(res, word)
	}

	return res
}

func suggestWords(words wordSet, c *choices) []string {
	// Deep copy the wordSet as it may get mutated.
	copy := make(wordSet)
	for word := range words {
		copy[word] = true
	}

	posMap, anyMap := populateMaps(copy)
	solver := &solver{
		choices: c,
		posMap:  posMap,
		anyMap:  anyMap,
		words:   copy,
	}
	return solver.suggestWords()
}

type wireChoices struct {
	KnownInPosition    map[int]string
	KnownOutOfPosition string
	KnownBad           string
}

func main() {
	port := flag.Int("port", 6500, "Port to listen on.")
	wordListPath := flag.String("word_list_path", "", "List of candidate words. One word per line.")
	flag.Parse()

	words, err := readWords(*wordListPath)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/suggest", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for the preflight request
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Set CORS headers for the main request.
		w.Header().Set("Access-Control-Allow-Origin", "*")

		var wChoices wireChoices
		if err := json.NewDecoder(r.Body).Decode(&wChoices); err != nil {
			help := `Failed to unmarshal the application/json request. Expected Schema:
	"KnownInPosition": map[int]string
	"KnownOutOfPosition": string
	"KnownBad": string
`
			http.Error(w, fmt.Sprintf("%s %v", help, err.Error()), http.StatusBadRequest)
			return
		}

		// Convert wire data to the application data.
		c := &choices{
			knownInPosition: make(map[int]byte),
		}

		knownGood := make(map[byte]bool)
		for k, v := range wChoices.KnownInPosition {
			c.knownInPosition[k] = byte(v[0])
			knownGood[v[0]] = true
		}
		for _, v := range wChoices.KnownOutOfPosition {
			c.knownOutOfPosition = append(c.knownOutOfPosition, byte(v))
			knownGood[byte(v)] = true
		}
		for _, v := range wChoices.KnownBad {
			c.knownBad = append(c.knownBad, byte(v))
			if _, found := knownGood[byte(v)]; found {
				w.Write([]byte(fmt.Sprintf("Conflicting configuration with letter %q.", v)))
				return
			}
		}

		results := suggestWords(words, c)
		if results == nil {
			w.Write([]byte("Unable to find any words."))
			return
		}

		b, err := json.Marshal(results)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write(b)
	})

	httpEndPoint := fmt.Sprintf(":%d", *port)
	fmt.Printf("Starting Go server on %s\n", httpEndPoint)
	if err := http.ListenAndServe(httpEndPoint, mux); err != nil {
		log.Fatal(err)
	}
}
