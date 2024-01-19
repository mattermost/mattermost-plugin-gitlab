import {beforeAll, describe, expect, it, jest} from "@jest/globals";
import {formatTimeSince} from "./date_utils";


describe('formatTimeSince should work as expected', () => {

    beforeAll(() => {
        const mockCurrentDate = new Date("December 17, 1995 10:11:00");
        jest.spyOn(Date, 'now').mockReturnValue(mockCurrentDate.getTime());
    })

    it.each([
        {
            dateString: 'December 17, 1995 10:10:30',
            output: '30 seconds'
        },
        {
            dateString: 'December 17, 1995 10:10:59',
            output: '1 second'
        },
        {
            dateString: 'December 17, 1995 09:41:00',
            output: '30 minutes'
        },
        {
            dateString: 'December 17, 1995 10:10:00',
            output: '1 minute'
        },
        {
            dateString: 'December 17, 1995 06:11:00',
            output: '4 hours'
        },
        {
            dateString: 'December 17, 1995 09:11:00',
            output: '1 hour'
        },
        {
            dateString: 'December 13, 1995 10:11:00',
            output: '4 days'
        },
        {
            dateString: 'December 16, 1995 10:11:00',
            output: '1 day'
        },
    ])('should show time since current time in proper format', ({dateString, output}) => {
        const result = formatTimeSince(dateString);
        expect(result).toBe(output)
    });
})