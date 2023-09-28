package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mhdbashar/prediction-go/prediction"
    "google.golang.org/grpc"

	"os"
	"time"
    "github.com/xhit/go-str2duration/v2"
	"strconv"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"


)

type PromMetaData struct{
	promAdd		 string
	history      string
	stepDuration string
	query		string
}	


func main() {

    // grpc client
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    

	//example of promdata
	promdata1 := PromMetaData{promAdd: "http://10.10.10.80:30909",
	history:"1d",
	stepDuration:"1m",
	query:"sum(rate(nginx_ingress_controller_requests{path='/fib'}[1m]))*60*1",
	}

	for {
		
    //prom query
	result := promdata1.doQuery()
	// fmt.Printf("Result:\n%v\n",result)
    
    out, err :=parsePrometheusResult(result)
    
    // fmt.Println(out)

	client := prediction.NewPredictionServiceClient(conn)

	requestData := &prediction.PredictionRequest{
		MicorserviceName: "testMicroService",
  		Measurements: out,
  		History: promdata1.history,
        StepDuration: promdata1.stepDuration,
  		PredictVerticalWindow: int32(1),
  		PredictHorizontalWindow: int32(8),
    }
    response, err := client.ProcessData(context.Background(), requestData )
    if err != nil {
        log.Fatalf("Error calling ProcessData: %v", err)
    }

    fmt.Println("Response: ", response.Result)
	//sleep
	duration := 60 * time.Second
	time.Sleep(duration)
	}

}

func (pmeta *PromMetaData) doQuery () (model.Value) {
    promclient, err := api.NewClient(api.Config{
		Address: pmeta.promAdd,
	})
	if err != nil {
		fmt.Printf("Error creating promclient: %v\n", err)
		os.Exit(1)
	}

	v1api := v1.NewAPI(promclient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

    currentTime := time.Now()
    hist,err := str2duration.ParseDuration(pmeta.history)
    if err != nil {
        fmt.Errorf("hist parsing error %w", err)
    }
    step,err := str2duration.ParseDuration(pmeta.stepDuration)
    if err != nil {
        fmt.Errorf("step parsing error %w", err)
    }

	r := v1.Range{
		Start: currentTime.Add(-hist),
		End:   currentTime,
		Step:  step,
	}
	result, warnings, err := v1api.QueryRange(ctx, pmeta.query, r)
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	return result
}

func parsePrometheusResult(result model.Value) (out []float64, err error) {
    switch result.Type() {
	case model.ValVector:
		if res, ok := result.(model.Vector); ok {
			for _, val := range res {
				if err != nil {
					return nil, err
				}
				out = append(out, float64(val.Value))
			}
		}
	case model.ValMatrix:
		if res, ok := result.(model.Matrix); ok {
			for _, val := range res {
				for _, v := range val.Values {
					if err != nil {
						return nil, err
					}

					out = append(out, float64(v.Value))
				}
			}
		}
	case model.ValScalar:
		if res, ok := result.(*model.Scalar); ok {
			if err != nil {
				return nil, err
			}

			out = append(out, float64(res.Value))
		}
	case model.ValString:
		if res, ok := result.(*model.String); ok {
			if err != nil {
				return nil, err
			}

			s, err := strconv.ParseFloat(res.Value, 64)
			if err != nil {
				return nil, err
			}

			out = append(out,s)
		}
	default:
		return nil, err
	}

	return out, nil
}

