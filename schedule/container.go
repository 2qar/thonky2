package schedule

import "github.com/bigheadgeorge/spreadsheet"

// Container stores cells in a format that fits each activity time every day.
type Container [7][6]*spreadsheet.Cell

// Values returns the string values of each cell
func (c *Container) Values() [7][6]string {
	var values [7][6]string
	for i, row := range c {
		for j, cell := range row {
			values[i][j] = cell.Value
		}
	}
	return values
}

// Fill fills a cell container with cells on a sheet starting at a given row and column.
func (c *Container) Fill(sheet *spreadsheet.Sheet, row, col int) {
	rowMax := row + 7
	colMax := col + 6
	for i := row; i < rowMax; i++ {
		for j := col; j < colMax; j++ {
			c[i-2][j-2] = &sheet.Rows[i][j]
		}
	}
}
