package main

import (
  "fmt"
  "time"
  "errors"
  "strings"
  "bytes"
  "log"
  "io/ioutil"
  "net/http"
)

// Base URL.
const BaseUrl = "https://www.service.bund.de/"

// Category: "IT und Telekommunikation".
const Category = "itundtelekommunikation"

// Job offer.
type JobOffer struct {
  description string
  employer string
  endDate time.Time
  url string
}

// Print job offer.
func (jobOffer *JobOffer) Print(idx int) {
  fmt.Printf("%d:\r\n", idx)
  fmt.Printf("\tDescription: %s\r\n", jobOffer.description)
  fmt.Printf("\tEmployer: %s\r\n", jobOffer.employer)
  fmt.Printf("\tEnd date: %s\r\n", jobOffer.endDate)
  fmt.Printf("\tURL: %s%s\r\n\r\n", BaseUrl, jobOffer.url)
}

// Parse job offer.
func parseJobOffer(s []byte, loc *time.Location) (*JobOffer, error) {
  // Example:
  // <a href="IMPORTE/Stellenangebote/editor/Technische-Universitaet-Muenchen/2020/04/3380505.html;jsessionid=8A12F82096E6667B7513EFA19A66F93C.2_cid296?nn=4642046&amp;type=0&amp;searchResult=true"
  //    title="Zur Detailseite 'Softwareentwickler/-innen (m/w/d) für Open-Source Software'">
  //   <div aria-labelledby="title">
  //     <h3>
  //       <em>Stellenbezeichnung</em>
  //       Soft­wa­re­ent­wick­ler/-in­nen (m/w/d) für Open-Sour­ce Soft­ware
  //     </h3>
  //     <p>
  //       <em>Arbeitgeber</em>
  //       Uni­ver­si­täts­bi­blio­thek der Tech­ni­schen Uni­ver­si­tät Mün­chen
  //     </p>
  //   </div>
  //   <div aria-labelledby="date">
  //     <p><em>Veröffentlicht</em> 09.04.20</p>
  //   </div>
  //   <div aria-labelledby="location">
  //     <p><em>Bewerbungsfrist</em> 07.05.20</p>
  //   </div>
  // </a>

  // Search end of the URL.
  end := bytes.IndexAny(s[9:], ";\"")

  // If the end of the URL was not found...
  if end == -1 {
    return nil, errors.New("End of URL not found")
  }

  // Extract URL.
  url := s[9 : 9 + end]

  // Search description.
  begin := bytes.Index(s, []byte("<em>Stellenbezeichnung</em>"))

  // If the description was not found...
  if begin == -1 {
    return nil, errors.New("Description not found")
  }

  begin += 27 // len("<em>Stellenbezeichnung</em>")

  // Search end of description.
  end = bytes.Index(s[begin:], []byte("</h3>"))

  // If the end of the description was not found...
  if end == -1 {
    return nil, errors.New("End of description not found")
  }

  // Extract description.
  description := bytes.ReplaceAll(bytes.Trim(s[begin : begin + end], " \t\r\n"),
                                  []byte("­"),
                                  []byte(""))

  // Search employer.
  begin = bytes.Index(s, []byte("<em>Arbeitgeber</em>"))

  // If the employer was not found...
  if begin == -1 {
    return nil, errors.New("Employer not found")
  }

  begin += 20 // len("<em>Arbeitgeber</em>")

  // Search end of employer.
  end = bytes.Index(s[begin:], []byte("</p>"))

  // If the end of the employer was not found...
  if end == -1 {
    return nil, errors.New("End of employer not found")
  }

  // Extract employer.
  employer := bytes.ReplaceAll(bytes.Trim(s[begin : begin + end], " \t\r\n"),
                               []byte("­"),
                               []byte(""))


  // Search end date.
  begin = bytes.Index(s, []byte("<em>Bewerbungsfrist</em>"))

  // If the end date was not found...
  if begin == -1 {
    return nil, errors.New("End date not found")
  }

  begin += 24 // len("<em>Bewerbungsfrist</em>")

  // Search end of end date.
  end = bytes.Index(s[begin:], []byte("</p>"))

  // If the end of the end date was not found...
  if end == -1 {
    return nil, errors.New("End of end date not found")
  }

  // Extract end date.
  dateStr := bytes.Trim(s[begin : begin + end], " \t\r\n")

  // Parse date.
  endDate, err := time.ParseInLocation("02.01.06", string(dateStr), loc)

  // If the date is invalid...
  if err != nil {
    return nil, fmt.Errorf("Invalid end date '%s' (error: '%s')", dateStr, err)
  }

  return &JobOffer{string(description),
                   string(employer),
                   endDate.Add((23 * time.Hour) +
                               (59 * time.Minute) +
                               (59 * time.Second)),
                   string(url)},
         nil
}

func main() {
  // Load location.
  loc, _ := time.LoadLocation("Europe/Berlin")

  // Compose URL.
  url := BaseUrl +
         "Content/DE/Stellen/Suche/Formular.html?cl2Categories_Taetigkeitsfeld=taetigkeitsfeld-" +
         Category +
         "&resultsPerPage=100"

  // Cookies.
  var cookies []*http.Cookie

  // Referer.
  var referer string

  count := 0

  for {
    // Create HTTP request.
    req, err := http.NewRequest("GET", url, nil)

    // Error?
    if err != nil {
      log.Fatal(err)
    }

    // Add cookies.
    for _, cookie := range cookies {
      // Add cookie.
      req.AddCookie(cookie)
    }

    // If there is referer...
    if len(referer) > 0 {
      req.Header.Set("Referer", referer)
    }

    client := &http.Client{}

    // Make HTTP request.
    resp, err := client.Do(req)

    // Error?
    if err != nil {
      log.Fatal(err)
    }

    defer resp.Body.Close()

    // Read the whole response.
    page, err := ioutil.ReadAll(resp.Body)

    // Error?
    if err != nil {
      log.Fatal(err)
    }

    // Save cookies.
    cookies = resp.Cookies()

    // Search begin of the results.
    begin := bytes.Index(page, []byte("<ul class=\"result-list\">"))

    // If no results were found...
    if begin == -1 {
      log.Fatal("Results not found.\n")
    }

    // Search end of the results.
    end := bytes.Index(page[begin:], []byte("</ul>"))

    // If the end of the results was not found...
    if end == -1 {
      log.Fatal("End of the results not found\n")
    }

    results := page[begin : begin + end]

    begin = 0

    // Parse job offers.
    for begin != -1 {
      // Search begin of the next job offer.
      idx := bytes.Index(results[begin:], []byte("<a href="))

      // If there are no more job offers...
      if idx == -1 {
        break
      }

      begin += idx

      // Search end of the next job offer.
      end = bytes.Index(results[begin:], []byte("</a>"))

      // If the end of the job offer was not found...
      if end == -1 {
        break
      }

      end += begin

      // Parse job offer.
      jobOffer, err := parseJobOffer(results[begin : end], loc)

      // Error?
      if err != nil {
        log.Fatal(err)
      }

      count++

      // Print job offer.
      jobOffer.Print(count)

      begin = end
    }

    // Get URL of the next page (if any).
    begin = bytes.Index(page, []byte("class=\"next\""))

    // If there are no more pages...
    if begin == -1 {
      break
    }

    end = bytes.Index(page[begin:], []byte("</li>"))

    if end == -1 {
      break
    }

    next := page[begin : begin + end]

    begin = bytes.Index(next, []byte("<a href=\""))

    if begin == -1 {
      break
    }

    begin += 9 // len("<a href=\"")

    end = bytes.Index(next[begin:], []byte("\""))

    if end == -1 {
      break
    }

    // Save previous URL as the referer.
    referer = url

    url = strings.ReplaceAll(BaseUrl + string(next[begin : begin + end]),
                             "amp;",
                             "")

    // Do not make requests too fast.
    time.Sleep(3 * time.Second)
  }
}
