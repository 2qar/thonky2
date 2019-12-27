package schedule

import "github.com/bigheadgeorge/spreadsheet"

// Container stores cells in a format that fits each activity time every day.
type Container [][]*spreadsheet.Cell

// Values returns the string values of each cell
func (c Container) Values() [][]string {
	values := make([][]string, len(c))
	for i, row := range c {
		rowValues := make([]string, len(c[0]))
		for j, cell := range row {
			rowValues[j] = cell.Value
		}
		values[i] = rowValues
	}
	return values
}

// Fill fills a cell container with cells on a sheet starting at a given row and column.
func (c Container) Fill(sheet *spreadsheet.Sheet, rowStart, rows, colStart, cols int) {
	values := make([][]*spreadsheet.Cell, rows)
	for i := rowStart; i < rowStart+rows; i++ {
		rowValues := make([]*spreadsheet.Cell, cols)
		for j := colStart; j < colStart+cols; j++ {
			rowValues[j-2] = &sheet.Rows[i][j]
		}
		values[i-2] = rowValues
	}
	c = values
}
