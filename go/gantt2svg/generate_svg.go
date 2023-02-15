package gantt2svg

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	gonggantt_models "github.com/fullstack-lang/gonggantt/go/models"

	gongsvg_models "github.com/fullstack-lang/gongsvg/go/models"
)

// type of the singloton for interception of gantt commit in order to generate
// the svg
type GanttToSVGTranformer struct {
}

var GanttToSVGTranformerSingloton GanttToSVGTranformer

func (GanttToSVGTranformer *GanttToSVGTranformer) GenerateSvg(stage *gonggantt_models.StageStruct) {

	// remove all gongsvg stage/repo
	gongsvg_models.Stage.Checkout()
	gongsvg_models.Stage.Reset()
	gongsvg_models.Stage.Commit()

	if len(*gonggantt_models.GetGongstructInstancesSet[gonggantt_models.Gantt]()) != 1 {
		log.Printf("It is supposed to have only one gantt chart")
		return
	}

	var ganttToRender *gonggantt_models.Gantt
	for gantt := range *gonggantt_models.GetGongstructInstancesSet[gonggantt_models.Gantt]() {
		ganttToRender = gantt
	}
	ganttToRender.ComputeStartAndEndDate()

	//
	// SVG
	//
	svg := new(gongsvg_models.SVG).Stage()
	svg.Name = "New Gantt Chart"
	svg.Display = true

	//
	// Time Line
	//
	// creates a tick lane
	LaneHeight := ganttToRender.LaneHeight
	RatioBarToLaneHeight := ganttToRender.RatioBarToLaneHeight
	barHeigth := LaneHeight * RatioBarToLaneHeight
	YTopMargin := ganttToRender.YTopMargin
	yTimeLine := LaneHeight*float64(len(*gonggantt_models.GetGongstructInstancesSet[gonggantt_models.Lane]())) + YTopMargin

	XLeftText := ganttToRender.XLeftText
	TextHeight := ganttToRender.TextHeight

	XLeftLanes := ganttToRender.XLeftLanes
	XRightMargin := ganttToRender.XRightMargin

	timeLine := new(gongsvg_models.Line).Stage()
	timeLine.Name = "Time Line"
	timeLine.X1 = XLeftLanes
	timeLine.Y1 = yTimeLine
	timeLine.X2 = XRightMargin
	timeLine.Y2 = yTimeLine

	timeLine.Color = ganttToRender.TimeLine_Color
	timeLine.FillOpacity = ganttToRender.TimeLine_FillOpacity
	timeLine.Stroke = ganttToRender.TimeLine_Stroke
	timeLine.StrokeWidth = ganttToRender.TimeLine_StrokeWidth

	svg.Lines = append(svg.Lines, timeLine)

	//
	// tick lines
	//

	//
	// Dates
	//

	// Date offset. Where is the date below the TimeLine
	DateYOffset := 20.0

	// put a date for every year
	for year := ganttToRender.ComputedStart.Year(); year <= ganttToRender.ComputedEnd.Year(); year++ {
		yearTime := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		durationBetweenYearAndGanttStart := yearTime.Sub(ganttToRender.ComputedStart)

		durationBetweenYearStartAndGanttStartRelativeToGanttDuration :=
			float64(durationBetweenYearAndGanttStart) / float64(ganttToRender.ComputedEnd.Sub(ganttToRender.ComputedStart))

		yearText := new(gongsvg_models.Text).Stage()
		yearText.Name = fmt.Sprintf("%d", year)
		yearText.Content = yearText.Name
		yearText.X = XLeftLanes + (XRightMargin-XLeftLanes)*durationBetweenYearStartAndGanttStartRelativeToGanttDuration
		yearText.Y = yTimeLine + DateYOffset
		yearText.Color = "black"
		yearText.FillOpacity = 1.0
		svg.Texts = append(svg.Texts, yearText)

		// log.Printf("year %d", year)
		//
		// draw the line
		//
		lineForYear := new(gongsvg_models.Line).Stage()
		lineForYear.Name = yearText.Name
		svg.Lines = append(svg.Lines, lineForYear)
		lineForYear.X1 = yearText.X
		lineForYear.X2 = lineForYear.X1
		lineForYear.Y1 = YTopMargin
		lineForYear.Y2 = yTimeLine
		lineForYear.Stroke = "black"
		lineForYear.StrokeWidth = 0.25
		lineForYear.StrokeDashArray = "4 4"
	}

	//
	// Lanes
	//
	currentY := YTopMargin

	mapLane_TextY := make(map[*gonggantt_models.Lane]float64, 0)
	mapLane_TopY := make(map[*gonggantt_models.Lane]float64, 0)

	//
	// Sort Lanes according to the Order
	//
	sort.Slice(ganttToRender.Lanes, func(i, j int) bool {
		return ganttToRender.Lanes[i].Order < ganttToRender.Lanes[j].Order
	})

	laneIndex := 0

	// prepare a map of bar to barSVG
	mapBar_BarSVG := make(map[*gonggantt_models.Bar]*gongsvg_models.Rect)
	for _, lane := range ganttToRender.Lanes {

		laneSVG := new(gongsvg_models.Rect).Stage()
		svg.Rects = append(svg.Rects, laneSVG)
		laneSVG.Name = lane.Name
		laneSVG.X = XLeftLanes
		laneSVG.Y = currentY
		mapLane_TopY[lane] = currentY

		laneSVG.Width = XRightMargin - XLeftLanes
		laneSVG.Height = LaneHeight

		laneSVG.Color = "grey"
		laneSVG.FillOpacity = 0.1

		if laneIndex%2 == 0 {
			laneSVG.Color = "black"
		}
		laneIndex = laneIndex + 1
		laneSVG.StrokeWidth = 1.5

		laneText := new(gongsvg_models.Text).Stage()
		laneText.Name = lane.Name
		laneText.Content = laneText.Name
		laneText.X = XLeftText
		laneText.Y = currentY + LaneHeight/2.0 + TextHeight/2.0
		mapLane_TextY[lane] = laneText.Y
		laneText.Color = "black"
		laneText.FillOpacity = 1.0
		svg.Texts = append(svg.Texts, laneText)

		//
		// Bar
		//
		for _, bar := range lane.Bars {
			barSVG := new(gongsvg_models.Rect).Stage()
			mapBar_BarSVG[bar] = barSVG
			svg.Rects = append(svg.Rects, barSVG)
			barSVG.Name = bar.Name

			var barToDisplay gonggantt_models.Bar
			barToDisplay = *bar

			// if start and end dates of the gantt are set manualy, then
			// reset start and end dates of the bar to display
			if ganttToRender.UseManualStartAndEndDates {
				if bar.Start.Before(ganttToRender.ManualStart) {
					barToDisplay.Start = ganttToRender.ManualStart
				}
				if bar.End.After(ganttToRender.ManualEnd) {
					barToDisplay.End = ganttToRender.ManualEnd
				}
			}

			durationFromGanttStartToBarStart := barToDisplay.Start.Sub(ganttToRender.ComputedStart)
			durationBetweenBarStartAndGanttStartRelativeToGanttDuration :=
				float64(durationFromGanttStartToBarStart) / float64(ganttToRender.ComputedEnd.Sub(ganttToRender.ComputedStart))
			// log.Printf("Duration is %f", durationBetweenBarStartAndGanttStartRelativeToGanttDuration)

			durationFromBarEndAndBarStart := barToDisplay.End.Sub(barToDisplay.Start)
			durationBetweenBarEndAndBarStartRelativeToGanttDuration :=
				float64(durationFromBarEndAndBarStart) / float64(ganttToRender.ComputedEnd.Sub(ganttToRender.ComputedStart))
			// log.Printf("Relative Duration is %f", durationBetweenBarEndAndBarStartRelativeToGanttDuration)

			barSVG.X = XLeftLanes + (XRightMargin-XLeftLanes)*durationBetweenBarStartAndGanttStartRelativeToGanttDuration
			barSVG.Y = currentY + (LaneHeight-barHeigth)/2.0
			barSVG.Height = barHeigth
			barSVG.Width = (XRightMargin - XLeftLanes) * durationBetweenBarEndAndBarStartRelativeToGanttDuration

			barSVG.Color = "blue"
			if bar.OptionnalColor != "" {
				barSVG.Color = bar.OptionnalColor
			}
			barSVG.FillOpacity = 0.1
			if bar.FillOpacity != 0.0 {
				barSVG.FillOpacity = bar.FillOpacity
			}
			barSVG.Stroke = "blue"
			if bar.OptionnalStroke != "" {
				barSVG.Stroke = bar.OptionnalStroke
			}
			barSVG.StrokeWidth = 0.5
			if bar.StrokeWidth != 0.0 {
				barSVG.StrokeWidth = bar.StrokeWidth
			}
			if bar.StrokeDashArray != "" {
				barSVG.StrokeDashArray = bar.StrokeDashArray
			}

			// bar text
			barText := new(gongsvg_models.Text).Stage()
			barText.Name = bar.Name
			barText.Content = barText.Name
			barText.X = barSVG.X + XLeftText
			barText.Y = currentY + LaneHeight/2.0 + TextHeight/2.0
			barText.Color = "black"
			barText.FillOpacity = 1.0
			svg.Texts = append(svg.Texts, barText)
		}

		currentY = currentY + LaneHeight

	}

	//
	// Milestones
	//
	for _, milestone := range ganttToRender.Milestones {

		if ganttToRender.UseManualStartAndEndDates &&
			(milestone.Date.Before(ganttToRender.ManualStart) ||
				milestone.Date.After(ganttToRender.ManualEnd)) {
			continue
		}
		durationBetweenMilestoneAndGanttStart := milestone.Date.Sub(ganttToRender.ComputedStart)

		durationBetweenMilestoneAndGanttStartString := durationBetweenMilestoneAndGanttStart.String()
		_ = durationBetweenMilestoneAndGanttStartString

		durationBetweenMilestoneAndGanttStartRelativeToGanttDuration :=
			float64(durationBetweenMilestoneAndGanttStart) / float64(ganttToRender.ComputedEnd.Sub(ganttToRender.ComputedStart))

		//
		// draw the line
		//
		lineX := XLeftLanes + (XRightMargin-XLeftLanes)*durationBetweenMilestoneAndGanttStartRelativeToGanttDuration
		if milestone.DisplayVerticalBar {
			line := new(gongsvg_models.Line).Stage()
			line.Name = milestone.Name
			svg.Lines = append(svg.Lines, line)
			line.X1 = lineX
			line.X2 = line.X1
			line.Y1 = YTopMargin
			line.Y2 = yTimeLine
			line.Stroke = "black"
			line.StrokeWidth = 0.5
			line.StrokeDashArray = "2 2"
		}

		//
		// draw diamond
		//
		diamondWidth := 18.0
		for _, lanesToDisplayMilestone := range milestone.LanesToDisplayMilestoneUse {

			diamond := new(gongsvg_models.Rect).Stage()
			svg.Rects = append(svg.Rects, diamond)
			diamond.Name = milestone.Name
			diamond.X = lineX - diamondWidth/2.0
			diamond.Y = mapLane_TextY[lanesToDisplayMilestone.Lane] - diamondWidth + LaneHeight/2.0
			diamond.Width = diamondWidth
			diamond.Height = diamondWidth
			diamond.Color = "red"
			diamond.FillOpacity = 0.4
			diamond.Transform = fmt.Sprintf("rotate(%d %d %d)", 45, int64(diamond.X+diamondWidth/2.0), int64(diamond.Y+diamondWidth/2.0))

			// bar text
			milestoneText := new(gongsvg_models.Text).Stage()
			milestoneText.Name = milestone.Name
			milestoneText.Content = milestoneText.Name
			milestoneText.X = diamond.X + XLeftText + diamondWidth
			milestoneText.Y = diamond.Y + TextHeight/2.0 + 5 // manual fine tuning
			milestoneText.Color = "black"
			milestoneText.FillOpacity = 1.0
			svg.Texts = append(svg.Texts, milestoneText)
		}
	}

	//
	// Arrows, for each arrow, draw five lines,
	//
	// start : middle of the end of the "Start" bar on
	//
	for _, arrow := range ganttToRender.Arrows {

		startBar := mapBar_BarSVG[arrow.From]
		endBar := mapBar_BarSVG[arrow.To]

		generate_arrow(
			svg,
			startBar.X+startBar.Width,
			endBar.X,
			startBar.Y+barHeigth/2.0,
			endBar.Y+barHeigth/2.0,
			ganttToRender.ArrowLengthToTheRightOfStartBar,
			ganttToRender.ArrowTipLenght,
			arrow.Name,
			arrow.OptionnalColor,
			arrow.OptionnalStroke)
	}

	//
	// Groups of Lanes
	//
	for _, group := range ganttToRender.Groups {

		if len(group.GroupLanes) == 0 {
			continue
		}

		groupSVG := new(gongsvg_models.Rect).Stage()
		svg.Rects = append(svg.Rects, groupSVG)
		groupSVG.Name = group.Name

		// compute X from list of lane
		groupTopY := math.MaxFloat64
		groupBottomY := 0.0
		for _, lane := range group.GroupLanes {
			if groupTopY > mapLane_TopY[lane] {
				groupTopY = mapLane_TopY[lane]
			}
			if groupBottomY < (mapLane_TopY[lane] + ganttToRender.LaneHeight) {
				groupBottomY = mapLane_TopY[lane] + ganttToRender.LaneHeight
			}
		}

		groupSVG.X = 0
		groupSVG.Y = groupTopY
		groupSVG.Width = ganttToRender.XRightMargin - groupSVG.X
		groupSVG.Height = groupBottomY - groupTopY

		groupSVG.Stroke = ganttToRender.Group_Stroke
		groupSVG.StrokeWidth = ganttToRender.Group_StrokeWidth
		groupSVG.StrokeDashArray = ganttToRender.Group_StrokeDashArray

		// text
		groupText := new(gongsvg_models.Text).Stage()
		groupText.Name = group.Name
		groupText.Content = groupText.Name
		groupText.X = XLeftText
		groupText.Y = groupTopY + ganttToRender.DateYOffset
		groupText.Color = "blue"
		groupText.FillOpacity = 0.5
		svg.Texts = append(svg.Texts, groupText)
	}

	gongsvg_models.Stage.Commit()

	// log.Printf("Before Commit")
}
