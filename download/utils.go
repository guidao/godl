package download

import (
	"fmt"
)

type Chunk struct {
	sort  int
	start int64
	end   int64
}

func (this Chunk) Range() string {
	if this.end <= 0 {
		return fmt.Sprintf("bytes=%v-", this.start)
	}
	return fmt.Sprintf("bytes=%v-%v", this.start, this.end)
}

func SplitChunk(size int64, n int) []Chunk {
	var chunks []Chunk
	start := int64(0)
	length := size / int64(n)
	i := 0
	for start < size {
		end := start + length
		if end >= size {
			end = 0
		}
		chunks = append(chunks, Chunk{
			sort:  i,
			start: start,
			end:   end,
		})
		i++
		start = start + length + 1
	}
	return chunks
}
