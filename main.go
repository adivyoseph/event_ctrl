package main

import (
    "fmt"
  "strconv"
    mqtt "github.com/eclipse/paho.mqtt.golang"
 //   "log"
    "os"
 //   "time"
     "encoding/json"
     "github.com/gofiber/fiber/v2"
     "github.com/gofiber/template/handlebars"
)

type AdminMsg struct {
   Cmd string    `json:"cmd"`
   Rate    int         `json:"rate"`
//   Partner    string    `json:"partner"`
   Sequence int   `json:"seq"`              //add for debugging
}

// status/discovery message
type StatusMsg struct {
    Name string    `json:"name"`             //POD instance name
    Type    string    `json:"type"`               // always status
    State    string    `json:"state"`             //idle or run
    Rate    int           `json:"rate"`                // get requests per second to issue
    Partner    string    `json:"partner"` // place holder
     Lost  int    `json:"lost"`                          //sequence numbers not returned
     Latency  int    `json:"latency"`          //total round trip lantency 
     Sequence int   `json:"seq"`              //status sequency number, used to detect restarts
}

var discovery = make(map[int]StatusMsg, 256)   //map index is agent id or POD name


type GlobalConfig struct {
    MqttHost    string
    MqttUser    string
    MqttPswd  string

}

var globalConfig = GlobalConfig{
    MqttHost: "localhost",
    MqttUser:    "martin",
    MqttPswd: "martin",
}

var state = 0
var grate = 1000
var gAdminSeq = 0


func brokerPubAdmin () {
    var broker = globalConfig.MqttHost
    var port = 1883

    opts := mqtt.NewClientOptions()
    opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
    opts.SetClientID("go_event_admin")
    opts.SetUsername( globalConfig.MqttUser)
    opts.SetPassword( globalConfig.MqttPswd)
    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }

    msg := AdminMsg{}
        if state == 1{
               msg.Cmd = "start"
        } else {
            msg.Cmd = "stop"
        }
        msg.Rate = grate
        msg.Sequence = gAdminSeq 
        gAdminSeq ++


            payload, _  := json.Marshal(msg)

            token := client.Publish("admin", 0, false, payload)
            token.Wait()

            client.Disconnect(250)

}

//go routine to maintain event gemnerator inventory and state
func brokerSubscribeStatus () {
    var broker =   globalConfig.MqttHost
    var port = 1883

    opts := mqtt.NewClientOptions()
    opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
    opts.SetClientID("go_event_ctlr")
    opts.SetUsername( globalConfig.MqttUser)
    opts.SetPassword( globalConfig.MqttPswd)
    opts.SetDefaultPublishHandler( func(client mqtt.Client, msg mqtt.Message) {
          fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
          var statusMsg = StatusMsg{}
          err := json.Unmarshal(msg.Payload(),  &statusMsg )
          if err != nil {
              panic(err.Error())
          }
          if statusMsg.Type == "status"  {
              var disc = StatusMsg{}
              disc.Name = statusMsg.Name
              disc.Type = statusMsg.Type
              disc.State = statusMsg.State
              disc.Rate = statusMsg.Rate
              disc.Lost = statusMsg.Lost
              disc.Latency = statusMsg.Latency
              disc.Sequence = statusMsg.Sequence
              discovery[0] = disc
          }
          //else ignore
         fmt.Printf(" discovery  %s\n", discovery)
    })

    //opts.OnConnect = connectHandler
    //opts.OnConnectionLost = connectLostHandler
    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }

    topic := "status"
    token := client.Subscribe(topic, 1, nil)
    token.Wait()


    engine := handlebars.New("./views", ".hbs")

  app := fiber.New(fiber.Config{
    Views: engine,
  })
/*
  app.Get("/", func(c *fiber.Ctx) error {
    return c.Render("index", fiber.Map{
      "indextitle": "Kiwi World!",
    })
  })
*/

  app.Get("/", func(c *fiber.Ctx) error {

      fmt.Printf("Get  state %d\n", state)
		// Render index within layouts/main
      var buttonText = "Start"
      if state ==1 {
          buttonText = "Stop"
      }


		return c.Render("index", fiber.Map{
            "indextitle": "Kiwi World!",
			"Title": "kiwi, World!",
                "rateSet":  grate,
                "button": buttonText,
                   "table": discovery,
		}, "layouts/main")
	})


    app.Post("/", func(c *fiber.Ctx) error {
         fmt.Printf("Post  rate %s\n", c.FormValue("rate"))
         //Todo validate rate
         var buttonText = "Start"
         if state ==1 {
             state = 0
             buttonText = "Start"
         } else {
             state = 1
             buttonText = "Stop"
             grate, _ = strconv.Atoi(c.FormValue("rate"))
         }

         brokerPubAdmin()

           return c.Render("index", fiber.Map{
               "indextitle": "Kiwi World!",
               "Title": "kiwi, World!",
                   "button": buttonText,
                       "rateSet":  grate,
                           "table": discovery,
           }, "layouts/main")
    })
  

  app.Listen(":8080")
}




func main() {
    //get environment settings
    var work string
    work = os.Getenv("CTRL_MQTT_HOST")
    if len(work) > 0 {
        fmt.Println("env CTRL_MQTT_HOST =", work)
        globalConfig.MqttHost = work
    }
    work = os.Getenv("CTRL_MQTT_USER")
    if len(work) > 0 {
        fmt.Println("env CTRL_MQTT_USER =", work)
        globalConfig.MqttUser = work
    }
    work = os.Getenv("CTRL_MQTT_PSWD");
    if len(work) > 0 {
        fmt.Println("env CTRL_MQTT_PSWD =", work)
        globalConfig.MqttPswd = work
    }


  go  brokerSubscribeStatus()
  for  {
  }
}


