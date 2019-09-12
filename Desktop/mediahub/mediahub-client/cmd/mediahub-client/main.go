package main

import(
	"flag"
	"encoding/json"
	"strings"
    "regexp"
    "strconv"
    "time"
    "net/http"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"bytes"
)

//job create and status api
var mediahub_api = "http://localhost:8082/mediahub/api/jobs"

//global logger
var logger = log.New()


//collect job response after sending a request to create a job
type responseBodyDecoderPost struct{
    Id string `json:"id"`
    Status string `json:"status"`
}

//collect job status
type responseBodyDecoderGet struct{
    Id string `json:"providerJobId"`
    Status string `json:"status"`
    Provider string `json:"providerName"`
    Progress int `json:"progress"`
}

//provider name, source url and Output presets needed for creating a job
type PresetConfigurationMediaHub struct {
	Provider string `json:"provider"`
	Source   string `json:"source"`
	Outputs []OutputConfiguration `json:"outputs"`
}

//output video presets
type VideoPresets struct {
	Height string `json:"height"`
	Width  string `json:"width"`
	Codec  string `json:"codec"`
	Bitrate string `json:"bitrate"`
	GopSize string `json:"gopSize"`
	GopMode string `json:"gopMode"`
	InterlaceMode string `json:"interlaceMode"`
}

//output audio presets
type AudioPresets struct {
	Codec string `json:"codec"`
	Bitrate string `json:"bitrate"`
}

//output transcode settings
type TranscodeSettings struct {
	Container string `json:"container"`
	RateControl string `json:"rateControl"`
	TwoPass bool   `json:"twoPass"`
	Video VideoPresets `json:"video"`
	Audio AudioPresets `json:"audio"`
}

//combining all output presets
type OutputConfiguration struct {
	TranscodeSettings TranscodeSettings `json:"transcodeSettings"`
	FileName string `json:"fileName"`
}

// function to poll for job status
func startPolling(jobID string) {
  for {
    
    //job status api
    url := mediahub_api + "/" + jobID;
    time.Sleep(10 * time.Second) // this sucks.. time.Tick
    
    resp, err := http.NewRequest("GET", url, nil) //method post
    client := &http.Client{}
    responseBody, err := client.Do(resp)
    if err != nil {
        logger.Fatalf(" [%v] Could not able to make a GET call to job status api : [%v] and ensure about url and it's accessibilty",time.Now().Format(time.RFC850),err)
    }
    defer resp.Body.Close()





    resp, err := http.Get(url)
    
    if err != nil {
        logger.Fatalf("[%v] Could not able to poll for job status: [%v] and ensure about url and it's accessibilty",time.Now().Format(time.RFC850),err)
    }

    defer resp.Body.Close()
    
    body, err := ioutil.ReadAll(resp.Body) // handle error

    if err != nil {
        logger.Fatalf("[%v] Could not able to read job status response body for job : %v",time.Now().Format(time.RFC850),err)
    }

    responseBodyDecoder := responseBodyDecoderGet{}

    err := json.Unmarshal(body, &responseBodyDecoder)
    
    if err != nil {
        logger.Fatal(" [%v] Failed to parse json response body from mediahub job status api: [%v] and ensure about url and it's accessibilty",time.Now().Format(time.RFC850),err)
    }
    logger.Infof("[%v] Job with id: %v has been completed with percentage completion: %v ",time.Now().Format(time.RFC850),jobID,responseBodyDecoder.Progress)
    

    if ( responseBodyDecoder.Progress >= 100){
        break
    }
    
    
  }
}

func main() {
	//a general json structure to send h.264 jobs
	job := PresetConfigurationMediaHub{
		Provider: "hybrik/bitmovin-new-sdk/mediaconvert",
		Source: "gs/s3",
		Outputs: []OutputConfiguration{			
			{
				TranscodeSettings: TranscodeSettings{
					Container: "mp4",
                	RateControl: "VBR",
                	TwoPass: false,
					Video: VideoPresets{
						Height: "640",
                    	Width: "360",
                    	Codec: "h264",
                    	Bitrate: "1000000",
                    	GopSize: "120",
                    	GopMode: "fixed",
                    	InterlaceMode: "progressive",
					},
					Audio: AudioPresets{
						Codec: "aac",
                    	Bitrate: "64000",
					},
				},
				FileName: "test.mp4",
			},
		},
	}

	//fetch source video file location, provider name, resolution values in pixels ex: 640*360, bandwidth and output file namefrom cli
    sourceUrl := flag.String("source", "gs/s3", "a source url")
    providerName := flag.String("provider", "hybrik/bitmovin-new-sdk/MediaConvert", "a provider name")
    resolutionValue := flag.String("resolution", "pixels*pixels", "a resolution in pixels")
    bandwidthValue := flag.String("bandwidth", "Mbps", "bandwidth in Mbps")
    outputFileName := flag.String("output", "FileName", "output video file name")
    flag.Parse()


    //split height and width values from resolution
    heightWidth := strings.Split(*resolutionValue, "x")

    //set values to the json payload from commandline
    job.Provider = *providerName
    job.Source = *sourceUrl
    job.Outputs[0].FileName = *outputFileName
    job.Outputs[0].TranscodeSettings.Video.Height = heightWidth[0]
    job.Outputs[0].TranscodeSettings.Video.Width = heightWidth[1]

    //convert bitrate from Mbps to bits per second
    reg, err := regexp.Compile("[^0-9]+")
    if err != nil {
        logger.Fatalf(" [%v] Could not able to find bandwidth value: %v\n"+"ensure to enter the right value for bandwidth",time.Now().Format(time.RFC850),err)
    }
    processedString := reg.ReplaceAllString(*bandwidthValue, "")

    //update bitrate value in bits per second
    bitrateValue, err := strconv.Atoi(processedString) //error handling
    bitrateValue = bitrateValue * 1000000
    job.Outputs[0].TranscodeSettings.Video.Bitrate = strconv.Itoa(bitrateValue)


    //create a json payload
    jobPayload, err := json.Marshal(job)
    if err != nil {
    	logger.Fatalf(" [%v] Could not able to generate a json payload for creation of job: %v\n"+ "ensure to check the logic for json payload creation",time.Now().Format(time.RFC850),err)
    }
 

    //Post request to create a job
    requestBody, err := http.NewRequest("POST", mediahub_api, bytes.NewBuffer(jobPayload)) //method post
    client := &http.Client{}
    responseBody, err := client.Do(requestBody)
    if err != nil {
        logger.Fatalf(" [%v] Could not able to make a POST call to create job: %v\n"+"ensure about url and it's accessibilty",time.Now().Format(time.RFC850),err)
    }
    defer responseBody.Body.Close()


    //collect response from post
    body, err := ioutil.ReadAll(responseBody.Body) //error handling
    if err != nil {
        logger.Fatal(" [%v] Failed to read response body from job create mediahub api: %v and ensure about url and it's accessibilty",time.Now().Format(time.RFC850),err)
    }


    responseBodyDecoder := responseBodyDecoderPost{}

    err := json.Unmarshal(body, &responseBodyDecoder)
    if err != nil {
        logger.Fatal(" [%v] Failed to parse response from job_create mediahub api: [%v] and ensure about url and it's accessibilty",time.Now().Format(time.RFC850),err)
    }

    logger.Infof(" [%v] Job has been created using MediaHub api with an unique ID: %v\n",time.Now().Format(time.RFC850),responseBodyDecoder.Id)
    startPolling(responseBodyDecoder.Id)



}


