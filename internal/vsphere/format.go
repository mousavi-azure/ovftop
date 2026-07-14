package vsphere

import "fmt"

func formatBytesGB(b int64) string {
	return fmt.Sprintf("%.1f GB", float64(b)/(1024*1024*1024))
}

func formatMHzGHz(mhz int32) string {
	return fmt.Sprintf("%.2f GHz", float64(mhz)/1000)
}

func formatMBasGB(mb int32) string {
	return fmt.Sprintf("%.1f GB", float64(mb)/1024)
}
