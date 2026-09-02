package general_command

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
)

// CodecVersion1 identifies binary encoded payloads vs legacy gob
const CodecVersion1 byte = 0x01

// EncodeTo encodes FSM_Command into a bytes.Buffer using zero-reflection binary encoding.
func (f *FSM_Command) EncodeTo(w *bytes.Buffer) error {
	w.WriteByte(CodecVersion1)

	var scratch [8]byte
	binary.LittleEndian.PutUint64(scratch[:], uint64(f.Now))
	w.Write(scratch[:8])

	w.WriteByte(byte(f.Type))

	switch f.Type {
	case RW:
		rwCmd, ok := f.CMD.(RWK_Command)
		if !ok {
			return fmt.Errorf("expected RWK_Command for RW type, got %T", f.CMD)
		}
		w.WriteByte(byte(rwCmd.Op))
		if rwCmd.Op == Write {
			wCmd, ok := rwCmd.CMD.(WK_Command)
			if !ok {
				return fmt.Errorf("expected WK_Command for Write op, got %T", rwCmd.CMD)
			}
			w.WriteByte(byte(wCmd.Op))
			writeString(w, wCmd.Key)
			writeBytes(w, wCmd.Value)
			writeString(w, wCmd.ColumnFamilyName)
			writeString(w, wCmd.ColumnFamilySector)
			binary.LittleEndian.PutUint32(scratch[:4], uint32(wCmd.TTL))
			w.Write(scratch[:4])
		} else if rwCmd.Op == Read {
			rCmd, ok := rwCmd.CMD.(RK_Command)
			if !ok {
				return fmt.Errorf("expected RK_Command for Read op, got %T", rwCmd.CMD)
			}
			w.WriteByte(byte(rCmd.Op))
			writeString(w, rCmd.Key)
			writeString(w, rCmd.KeyPattern)
			writeString(w, rCmd.Cursor)
			binary.LittleEndian.PutUint64(scratch[:8], uint64(rCmd.Limit))
			w.Write(scratch[:8])
			writeString(w, rCmd.ColumnFamilyName)
			writeString(w, rCmd.ColumnFamilySector)
			binary.LittleEndian.PutUint64(scratch[:8], uint64(rCmd.TTL))
			w.Write(scratch[:8])
		} else {
			return fmt.Errorf("unknown RW_Type: %v", rwCmd.Op)
		}
	case DDL_FC:
		ddlCmd, ok := f.CMD.(DDL_Command)
		if !ok {
			return fmt.Errorf("expected DDL_Command for DDL_FC type, got %T", f.CMD)
		}
		w.WriteByte(byte(ddlCmd.Op))
		writeString(w, ddlCmd.ColumnFamilyName)
	case MCL:
		mclCmd, ok := f.CMD.(MCLK_Command)
		if !ok {
			return fmt.Errorf("expected MCLK_Command for MCL type, got %T", f.CMD)
		}
		w.WriteByte(byte(mclCmd.Op))
	case REPOSITORY_COMMAND:
		typeName, jsonBytes, err := EncodeRepoCommand(f.CMD)
		if err != nil {
			return fmt.Errorf("failed to encode repository command: %w", err)
		}
		writeString(w, typeName)
		writeBytes(w, jsonBytes)
	default:
		return fmt.Errorf("unknown command type: %v", f.Type)
	}
	return nil
}

// DecodeFrom decodes FSM_Command from a byte slice.
func (f *FSM_Command) DecodeFrom(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty payload for FSM_Command")
	}

	if data[0] != CodecVersion1 {
		return gob.NewDecoder(bytes.NewReader(data)).Decode(f)
	}

	if len(data) < 10 {
		return fmt.Errorf("data too short for binary FSM_Command header: %d bytes", len(data))
	}

	offset := 1
	f.Now = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	f.Type = Command_Type(data[offset])
	offset++

	switch f.Type {
	case RW:
		if offset >= len(data) {
			return io.ErrUnexpectedEOF
		}
		rwOp := RW_Type(data[offset])
		offset++

		if rwOp == Write {
			if offset >= len(data) {
				return io.ErrUnexpectedEOF
			}
			wOp := W_Type(data[offset])
			offset++

			key, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			val, n, err := readBytes(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			cfName, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			cfSector, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			if offset+4 > len(data) {
				return io.ErrUnexpectedEOF
			}
			ttl := int(binary.LittleEndian.Uint32(data[offset : offset+4]))

			f.CMD = RWK_Command{
				Op: rwOp,
				CMD: WK_Command{
					Op:                 wOp,
					Key:                key,
					Value:              val,
					ColumnFamilyName:   cfName,
					ColumnFamilySector: cfSector,
					TTL:                ttl,
				},
			}
		} else if rwOp == Read {
			if offset >= len(data) {
				return io.ErrUnexpectedEOF
			}
			rOp := R_Type(data[offset])
			offset++

			key, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			keyPattern, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			cursor, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			if offset+8 > len(data) {
				return io.ErrUnexpectedEOF
			}
			limit := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
			offset += 8

			cfName, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			cfSector, n, err := readString(data[offset:])
			if err != nil {
				return err
			}
			offset += n

			if offset+8 > len(data) {
				return io.ErrUnexpectedEOF
			}
			ttl := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))

			f.CMD = RWK_Command{
				Op: rwOp,
				CMD: RK_Command{
					Op:                 rOp,
					Key:                key,
					KeyPattern:         keyPattern,
					Cursor:             cursor,
					Limit:              limit,
					ColumnFamilyName:   cfName,
					ColumnFamilySector: cfSector,
					TTL:                ttl,
				},
			}
		}
	case DDL_FC:
		if offset >= len(data) {
			return io.ErrUnexpectedEOF
		}
		ddlOp := DDL_FC_Type(data[offset])
		offset++

		cfName, _, err := readString(data[offset:])
		if err != nil {
			return err
		}

		f.CMD = DDL_Command{
			Op:               ddlOp,
			ColumnFamilyName: cfName,
		}
	case MCL:
		if offset >= len(data) {
			return io.ErrUnexpectedEOF
		}
		mclOp := MCL_Type(data[offset])

		f.CMD = MCLK_Command{
			Op: mclOp,
		}
	case REPOSITORY_COMMAND:
		typeName, n1, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n1

		jsonBytes, _, err := readBytes(data[offset:])
		if err != nil {
			return err
		}

		repoCmd, err := DecodeRepoCommand(typeName, jsonBytes)
		if err != nil {
			return fmt.Errorf("failed to decode repository command %s: %w", typeName, err)
		}
		f.CMD = repoCmd
	default:
		return fmt.Errorf("unknown command type: %v", f.Type)
	}
	return nil
}

// EncodeTo encodes Query_Command into a bytes.Buffer.
func (q *Query_Command) EncodeTo(w *bytes.Buffer) error {
	w.WriteByte(CodecVersion1)
	var scratch [8]byte
	binary.LittleEndian.PutUint64(scratch[:], uint64(q.Now))
	w.Write(scratch[:8])

	switch cmd := q.Command.(type) {
	case RK_Command:
		w.WriteByte(0) // Tag 0: RK_Command
		w.WriteByte(byte(cmd.Op))
		writeString(w, cmd.Key)
		writeString(w, cmd.KeyPattern)
		writeString(w, cmd.Cursor)
		binary.LittleEndian.PutUint64(scratch[:8], uint64(cmd.Limit))
		w.Write(scratch[:8])
		writeString(w, cmd.ColumnFamilyName)
		writeString(w, cmd.ColumnFamilySector)
		binary.LittleEndian.PutUint64(scratch[:8], uint64(cmd.TTL))
		w.Write(scratch[:8])
	case *RK_Command:
		if cmd != nil {
			w.WriteByte(0) // Tag 0: RK_Command
			w.WriteByte(byte(cmd.Op))
			writeString(w, cmd.Key)
			writeString(w, cmd.KeyPattern)
			writeString(w, cmd.Cursor)
			binary.LittleEndian.PutUint64(scratch[:8], uint64(cmd.Limit))
			w.Write(scratch[:8])
			writeString(w, cmd.ColumnFamilyName)
			writeString(w, cmd.ColumnFamilySector)
			binary.LittleEndian.PutUint64(scratch[:8], uint64(cmd.TTL))
			w.Write(scratch[:8])
		}
	case Repository_Command, *Repository_Command:
		w.WriteByte(1) // Tag 1: Repository_Command
		typeName, jsonBytes, err := EncodeRepoCommand(q.Command)
		if err != nil {
			return err
		}
		writeString(w, typeName)
		writeBytes(w, jsonBytes)
	default:
		return fmt.Errorf("unsupported query command type: %T", q.Command)
	}
	return nil
}

// DecodeFrom decodes Query_Command from a byte slice.
func (q *Query_Command) DecodeFrom(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty payload for Query_Command")
	}

	if data[0] != CodecVersion1 {
		return gob.NewDecoder(bytes.NewReader(data)).Decode(q)
	}

	if len(data) < 10 {
		return fmt.Errorf("data too short for Query_Command header: %d bytes", len(data))
	}

	offset := 1
	q.Now = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	tag := data[offset]
	offset++

	if tag == 0 { // RK_Command
		if offset >= len(data) {
			return io.ErrUnexpectedEOF
		}
		rOp := R_Type(data[offset])
		offset++

		key, n, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		keyPattern, n, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		cursor, n, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		if offset+8 > len(data) {
			return io.ErrUnexpectedEOF
		}
		limit := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
		offset += 8

		cfName, n, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		cfSector, n, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n

		if offset+8 > len(data) {
			return io.ErrUnexpectedEOF
		}
		ttl := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))

		q.Command = RK_Command{
			Op:                 rOp,
			Key:                key,
			KeyPattern:         keyPattern,
			Cursor:             cursor,
			Limit:              limit,
			ColumnFamilyName:   cfName,
			ColumnFamilySector: cfSector,
			TTL:                ttl,
		}
	} else if tag == 1 { // Repository_Command
		typeName, n1, err := readString(data[offset:])
		if err != nil {
			return err
		}
		offset += n1

		jsonBytes, _, err := readBytes(data[offset:])
		if err != nil {
			return err
		}

		repoCmd, err := DecodeRepoCommand(typeName, jsonBytes)
		if err != nil {
			return fmt.Errorf("failed to decode query command %s: %w", typeName, err)
		}
		q.Command = Repository_Command{CMD: repoCmd}
	} else {
		return fmt.Errorf("unknown tag in Query_Command: %d", tag)
	}
	return nil
}

func writeString(w *bytes.Buffer, s string) {
	var scratch [4]byte
	binary.LittleEndian.PutUint32(scratch[:], uint32(len(s)))
	w.Write(scratch[:4])
	w.WriteString(s)
}

func writeBytes(w *bytes.Buffer, b []byte) {
	var scratch [4]byte
	binary.LittleEndian.PutUint32(scratch[:], uint32(len(b)))
	w.Write(scratch[:4])
	w.Write(b)
}

func readString(data []byte) (string, int, error) {
	if len(data) < 4 {
		return "", 0, io.ErrUnexpectedEOF
	}
	l := int(binary.LittleEndian.Uint32(data[:4]))
	if len(data) < 4+l {
		return "", 0, io.ErrUnexpectedEOF
	}
	return string(data[4 : 4+l]), 4 + l, nil
}

func readBytes(data []byte) ([]byte, int, error) {
	if len(data) < 4 {
		return nil, 0, io.ErrUnexpectedEOF
	}
	l := int(binary.LittleEndian.Uint32(data[:4]))
	if len(data) < 4+l {
		return nil, 0, io.ErrUnexpectedEOF
	}
	res := make([]byte, l)
	copy(res, data[4 : 4+l])
	return res, 4 + l, nil
}
