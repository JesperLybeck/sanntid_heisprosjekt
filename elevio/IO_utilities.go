package elevio

import (
	"fmt"
)

func Handle_button_event(event Button_event, order_list *[]Elevator_order) {
	fmt.Println("Button event")
	*order_list = append(*order_list, Elevator_order{floor: event.floor})
}
