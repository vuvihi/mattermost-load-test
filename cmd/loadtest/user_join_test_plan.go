package main

// This test makes a few excessive calls, please simplify like user_simple

// The join test is to test joining as many users to a channel as
// possible. The Test plans do not restart after finishing.
import (
	"errors"
	"fmt"
	"time"

	l "github.com/mattermost/mattermost-load-test/lib"
	p "github.com/mattermost/mattermost-load-test/platform"
)

// UserJoinTestPlan - try out the test plan interface
type UserJoinTestPlan struct {
	id              int
	activityChannel chan<- l.Activity
	mm              p.Platform
}

// Generator sets up & exports the channels
func (tp UserJoinTestPlan) Generator(id int, activityChannel chan<- l.Activity) l.TestPlan {
	newPlan := new(UserJoinTestPlan)
	newPlan.id = id
	newPlan.activityChannel = activityChannel
	newPlan.mm = p.GeneratePlatform(Config.PlatformURL)
	return newPlan
}

// Start is a long running function that should only quit on error
func (tp *UserJoinTestPlan) Start() bool {

	defer tp.PanicCheck()

	userEmail := GeneratePlatformEmail(tp.id)
	userPassword := GeneratePlatformPass(tp.id)
	userName := GeneratePlatformUsername(tp.id)
	userFirst := GeneratePlatformFirst(tp.id)
	userLast := GeneratePlatformLast()

	// Ping Server
	_, err := tp.mm.PingServer()
	if err != nil {
		tp.registerLaunchFail()
		return tp.handleError(err, "Ping Failed", false)
	}

	// Login User
	err = tp.mm.Login(userEmail, userPassword)
	if err != nil {
		tp.registerLaunchFail()
		tp.handleError(err, "Login Failed", true)
		return false
	}

	tp.registerActive()

	// Update Good
	err = tp.mm.UpdateProfile(userFirst, userLast, userName)
	if err != nil {
		return tp.handleError(err, "Profile Update Failed", true)
	}

	// Initial Load
	err = tp.mm.InitialLoad()
	if err != nil {
		return tp.handleError(err, "Initial Load Failed", true)
	}

	// Team Lookup Load
	_, err = tp.mm.FindTeam(Config.TeamName, true)
	if err != nil {
		return tp.handleError(err, "Team Lookup Failed", true)
	}

	// Join Test Channel
	err = tp.mm.JoinChannel(Config.TestChannel)
	if err != nil {
		return tp.handleError(err, "Join Channel Failed", true)
	}

	tp.sendJoin()
	tp.registerInactive()
	return false
}

// Stop takes the result of start(), and can change return
// respond true if the thread should restart, false otherwise
func (tp *UserJoinTestPlan) Stop() {

}

// GlobalSetup will run before the test plan. It will spin up a basic test plan
// from the Generator and will not be reused
func (tp *UserJoinTestPlan) GlobalSetup() (err error) {
	Info.Println("Starting Global Setup")
	return nil
}

// PanicCheck will check for panics, used as a defer in test plan
func (tp *UserJoinTestPlan) PanicCheck() {
	if r := recover(); r != nil {
		if Error != nil {
			Error.Printf("ERROR ON WORKER: %v", r)
		} else {
			fmt.Printf("ERROR ON WORKER: %v", r)
		}
		switch x := r.(type) {
		case string:
			tp.handleError(errors.New(x), "Error caught unexpected (thread failed)", true)
		case error:
			tp.handleError(x, "Error caught unexpected (thread failed)", true)
		default:
			tp.handleError(errors.New("Unknown Panic"), "Error caught unexpected (thread failed)", true)
		}
	}
}

func (tp *UserJoinTestPlan) registerActive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusActive,
		ID:      tp.id,
		Message: "Thread active",
	}
}

func (tp *UserJoinTestPlan) registerInactive() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusInactive,
		ID:      tp.id,
		Message: "Thread inactive",
	}
}

func (tp *UserJoinTestPlan) registerLaunchFail() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusLaunchFailed,
		ID:      tp.id,
		Message: "Failed launch",
	}
}

func (tp *UserJoinTestPlan) handleError(err error, message string, notify bool) bool {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusError,
		ID:      tp.id,
		Message: message,
		Err:     err,
	}
	if notify {
		tp.registerInactive()
	}
	time.Sleep(time.Second * 5)
	return true
}

func (tp *UserJoinTestPlan) sendJoin() {
	tp.activityChannel <- l.Activity{
		Status:  l.StatusAction,
		ID:      tp.id,
		Message: fmt.Sprintf("User #%v joined channel", tp.id),
	}
}
