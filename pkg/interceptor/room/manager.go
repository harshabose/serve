package room

//
// type manager struct {
// 	rooms map[string]*room
// 	ctx   context.Context
// 	mux   sync.Mutex
// }
//
// func (rooms *manager) Process(msg *Message, conn *websocket.Conn, writer interceptor.Writer, reader interceptor.Reader) error {
// 	return msg.Payload.Process(msg.Header, rooms, conn, writer, reader)
// }
//
// func (payload *CreateRoom) Process(header message.Header, manager *manager, conn *websocket.Conn, writer interceptor.Writer, reader interceptor.Reader) error {
// 	if err := payload.Validate(); err != nil {
// 		return err
// 	}
//
// 	manager.mux.Lock()
// 	defer manager.mux.Unlock()
//
// 	owner := &connection{
// 		conn:   conn,
// 		Writer: writer,
// 		Reader: nil,
// 	}
//
// 	if _, exists := manager.rooms[payload.RoomID]; exists {
// 		return errors.New("room already exists")
// 	}
//
// 	room := createRoom(manager.ctx, owner, payload.ClientsToAllow, payload.CloseTime)
// 	manager.rooms[payload.RoomID] = room
//
// 	return nil
// }
//
// func (msg *JoinRoom) Process(header message.Header, manager *manager, conn *websocket.Conn, writer interceptor.Writer, reader interceptor.Reader) error {
// 	if err := msg.Validate(); err != nil {
// 		return err
// 	}
//
// 	manager.mux.Lock()
// 	defer manager.mux.Unlock()
//
// 	participant := &connection{
// 		id:     msg.ClientID,
// 		conn:   conn,
// 		Writer: writer,
// 		Reader: nil,
// 	}
//
// 	if room, exists := manager.rooms[msg.RoomID]; exists {
// 		if err := room.add(participant); err != nil {
// 			return err
// 		}
// 		// TODO: Send JoinRoomSuccessMessage to sender
// 		return nil
// 	}
// 	return errors.New("room does not exists")
// }
//
// func (msg *LeaveRoom) Process(manager *manager, conn *websocket.Conn, _ interceptor.Writer) error {
// 	if err := msg.Validate(); err != nil {
// 		return err
// 	}
//
// 	manager.mux.Lock()
// 	defer manager.mux.Unlock()
//
// 	participant := &connection{
// 		id:   msg.ClientID,
// 		conn: conn,
// 	}
//
// 	if room, exists := manager.rooms[msg.RoomID]; exists {
// 		if err := room.remove(participant); err != nil {
// 			return err
// 		}
// 		// TODO: Send LeaveRoomSuccessMessage to sender
// 		return nil
// 	}
// 	return errors.New("room does not exists")
// }
//
// func (msg *ChatSource) Process(manager *manager, conn *websocket.Conn, _ interceptor.Writer) error {
// 	if err := msg.Validate(); err != nil {
// 		return err
// 	}
//
// 	manager.mux.Lock()
// 	defer manager.mux.Unlock()
//
// 	sender := &connection{
// 		id:   msg.ClientID,
// 		conn: conn,
// 	}
//
// 	if room, exists := manager.rooms[msg.RoomID]; exists {
// 		if err := room.send(sender, msg.RecipientID, msg.Content); err != nil {
// 			return err
// 		}
// 		// TODO: Send ChatRoomSuccessMessage to sender
// 		return nil
// 	}
// 	return errors.New("room does not exists")
// }
