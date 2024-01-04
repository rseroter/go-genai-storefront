package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Define a struct to hold the data from your JSON file
type Record struct {
	ID          int
	Name        string
	Description string
	ImageURL    string
}

type UserPref struct {
	Name        string
	Preferences string
}

func main() {

	// Parse the HTML templates
	tmpl := template.Must(template.ParseFiles("home.html", "details.html"))

	//return the home page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		var recordType string
		var recordDataFile string
		var personId string

		//if a post-back from a change in record type or persona
		if r.Method == "POST" {
			// Handle POST request:
			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Error parsing form data", http.StatusInternalServerError)
				return
			}

			// Extract values from POST data
			recordType = r.FormValue("recordtype")
			recordDataFile = "data/" + recordType + ".json"
			personId = r.FormValue("person")

		} else {
			// Handle GET request (or other methods):
			// Load default values
			recordType = "property"
			recordDataFile = "data/property.json"
			personId = "person1" // Or any other default person
		}

		// Parse the JSON file
		data, err := os.ReadFile(recordDataFile)
		if err != nil {
			fmt.Println("Error reading JSON file:", err)
			return
		}

		var records []Record
		err = json.Unmarshal(data, &records)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}

		// Execute the template and send the results to the browser
		err = tmpl.ExecuteTemplate(w, "home.html", struct {
			RecordType string
			Records    []Record
			Person     string
		}{
			RecordType: recordType,
			Records:    records,
			Person:     personId,
		})
		if err != nil {
			fmt.Println("Error executing template:", err)
		}
	})

	//returns the details page using AI assistance
	http.HandleFunc("/details", func(w http.ResponseWriter, r *http.Request) {

		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			fmt.Println("Error parsing ID:", err)
			// Handle the error appropriately (e.g., redirect to error page)
			return
		}

		// Extract values from querystring data
		recordType := r.URL.Query().Get("recordtype")
		recordDataFile := "data/" + recordType + ".json"

		//declare recordtype map and extract selected entry
		typeMap := make(map[string]string)
		typeMap["property"] = "Create an improved home listing description that's seven sentences long and oriented towards a a person with these preferences:"
		typeMap["store"] = "Create an updated paragraph-long summary of this store item that's colored by these preferences:"
		typeMap["restaurant"] = "Create a two sentence summary for this menu item that factors in one or two of these preferences:"
		//get the preamble for the chosen record type
		aiPremble := typeMap[recordType]

		// Parse the JSON file
		data, err := os.ReadFile(recordDataFile)
		if err != nil {
			fmt.Println("Error reading JSON file:", err)
			return
		}

		var records []Record
		err = json.Unmarshal(data, &records)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}

		// Find the record with the matching ID
		var record Record
		for _, rec := range records {
			if rec.ID == id { // Assuming your struct has an "ID" field
				record = rec
				break
			}
		}

		if record.ID == 0 { // Record not found
			// Handle the error appropriately (e.g., redirect to error page)
			return
		}

		//get a reference to the persona
		person := "personas/" + (r.URL.Query().Get("person") + ".json")

		//retrieve preference data from file name matching person variable value
		preferenceData, err := os.ReadFile(person)
		if err != nil {
			fmt.Println("Error reading JSON file:", err)
			return
		}
		//unmarshal the preferenceData response into an UserPref struct
		var userpref UserPref
		err = json.Unmarshal(preferenceData, &userpref)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}

		//improve the message using Gemini
		ctx := context.Background()
		// Access your API key as an environment variable (see "Set up your API key" above)
		client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
		if err != nil {
			log.Fatal(err)
		}
		defer client.Close()

		// For text-only input, use the gemini-pro model
		model := client.GenerativeModel("gemini-pro")
		resp, err := model.GenerateContent(ctx, genai.Text(aiPremble+" "+userpref.Preferences+". "+record.Description))
		if err != nil {
			log.Fatal(err)
		}

		//parse the response from Gemini
		bs, _ := json.Marshal(resp.Candidates[0].Content.Parts[0])
		record.Description = string(bs)

		//execute the template, and pass in the record
		err = tmpl.ExecuteTemplate(w, "details.html", record)
		if err != nil {
			fmt.Println("Error executing template:", err)
		}
	})

	fmt.Println("Server listening on port 8080")
	fs := http.FileServer(http.Dir("./images"))
	http.Handle("/images/", http.StripPrefix("/images/", fs))
	http.ListenAndServe(":8080", nil)
}
