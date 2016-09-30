package ical

import (
	"time"
)

type CalendarItem struct {
	id string
	summary string
	description string
	priority int // 0..9 (O -> none, 1 -> highest, 9 -> lowest)
	percentComplete int
	createdDate time.Time
	modifiedDate time.Time
	completedDate time.Time
	startDate time.Time
	alarmDate time.Time
	sequence int
}

func (this *CalendarItem) SetId(v string) { this.id = v }
func (this *CalendarItem) Id() string {	return this.id }
func (this *CalendarItem) SetSummary(v string) { this.summary = v }
func (this *CalendarItem) Summary() string { return this.summary }
func (this *CalendarItem) SetDescription(v string) { this.description = v }
func (this *CalendarItem) Description() string { return this.description }
func (this *CalendarItem) SetPriority(v int) { this.priority = v }
func (this *CalendarItem) Priority() int { return this.priority }
func (this *CalendarItem) SetPercentComplete(v int) { this.percentComplete = v }
func (this *CalendarItem) PercentComplete() int { return this.percentComplete }
func (this *CalendarItem) SetCreatedDate(v time.Time) { this.createdDate = v }
func (this *CalendarItem) CreatedDate() time.Time { return this.createdDate }
func (this *CalendarItem) SetModifiedDate(v time.Time) { this.modifiedDate = v }
func (this *CalendarItem) ModifiedDate() time.Time { return this.modifiedDate }
func (this *CalendarItem) SetCompletedDate(v time.Time) { this.completedDate = v }
func (this *CalendarItem) CompletedDate() time.Time { return this.completedDate }
func (this *CalendarItem) SetStartDate(v time.Time) { this.startDate = v }
func (this *CalendarItem) StartDate() time.Time { return this.startDate }
func (this *CalendarItem) SetAlarmDate(v time.Time) { this.alarmDate = v }
func (this *CalendarItem) AlarmDate() time.Time { return this.alarmDate }
func (this *CalendarItem) SetSequence(v int) { this.sequence = v }
func (this *CalendarItem) Sequence() int { return this.sequence }
