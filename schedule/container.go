package schedule

import "github.com/bigheadgeorge/spreadsheet"

// container stores cells in a format that fits each day of the week.
type container [7][6]*spreadsheet.Cell

// Values returns the string values of each cell
func (c *container) Values() [7][6]string {
	var values [7][6]string
	for i, row := range c {
		for j, cell := range row {
			values[i][j] = cell.Value
		}
	}
	return values
}

// Fill fills a cell container with cells on a sheet starting at a given row and column.
func (c *container) Fill(sheet *spreadsheet.Sheet, row, col int) {
	rowMax := row + 7
	colMax := col + 6
	for i := row; row < rowMax; row++ {
		for j := col; col < colMax; col++ {
			c[i-row][j-col] = &sheet.Rows[i][j]
		}
	}
}
