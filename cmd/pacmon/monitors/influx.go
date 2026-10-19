package monitors

import (
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

func IoChannelToTileId(ioChannel int) int {
	return (ioChannel-1)/4 + 1
}

func makePoint(name string, timeNow time.Time) *write.Point {
	return influxdb2.NewPoint(name, nil, nil, timeNow)
}

func addTag(point *write.Point, key string, value uint8) {
	point.AddTag(key, strconv.Itoa(int(value)))
}

func addIoChannelTags(point *write.Point, ioChannel IoChannelKey) {
	addTag(point, "io_group", ioChannel.IoGroup)
	addTag(point, "io_channel", ioChannel.IoChannel)
	point.AddTag(
		"tile_id",
		strconv.Itoa(IoChannelToTileId(int(ioChannel.IoChannel))),
	)
}

func addChipTags(point *write.Point, chip ChipKey) {
	addIoChannelTags(
		point,
		IoChannelKey{IoGroup: chip.IoGroup, IoChannel: chip.IoChannel},
	)
	addTag(point, "chip", chip.ChipID)
}

func addChannelTags(point *write.Point, channel ChannelKey) {
	addChipTags(
		point,
		ChipKey{
			IoGroup:   channel.IoGroup,
			IoChannel: channel.IoChannel,
			ChipID:    channel.ChipID,
		},
	)
	addTag(point, "channel", channel.ChannelID)
}

func addDataStatusFields(
	point *write.Point,
	counts DataStatusCounts,
	timeDiff float64,
) {
	point.AddField("total", float64(counts.Total)/timeDiff)
	point.AddField("valid_parity", float64(counts.ValidParity)/timeDiff)
	point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
	point.AddField("downstream", float64(counts.Downstream)/timeDiff)
	point.AddField("upstream", float64(counts.Upstream)/timeDiff)
}

func addConfigStatusFields(
	point *write.Point,
	counts ConfigStatusCounts,
	timeDiff float64,
) {
	point.AddField("total", float64(counts.Total)/timeDiff)
	point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
	point.AddField(
		"downstream_read",
		float64(counts.DownstreamRead)/timeDiff,
	)
	point.AddField(
		"downstream_write",
		float64(counts.DownstreamWrite)/timeDiff,
	)
	point.AddField("upstream_read", float64(counts.UpstreamRead)/timeDiff)
	point.AddField("upstream_write", float64(counts.UpstreamWrite)/timeDiff)
}

func addFifoFields(
	point *write.Point,
	lessHalfFull, moreHalfFull, full uint,
	total float64,
) {
	point.AddField("less_half_full", float64(lessHalfFull)/total)
	point.AddField("more_half_full", float64(moreHalfFull)/total)
	point.AddField("full", float64(full)/total)
}
