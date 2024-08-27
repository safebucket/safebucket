"use client"

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger
} from "@/components/ui/dialog";
import {Button} from "@/components/ui/button";
import {PlusCircle} from "lucide-react";
import {Label} from "@/components/ui/label";
import {Input} from "@/components/ui/input";
import {Switch} from "@/components/ui/switch";
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select";
import {useState} from "react";
import {DatePickerDemo} from "@/components/datepicker";

export default function ShareFileDialog() {
    const [expiresAt, setExpiresAt] = useState(false)

    return (
        <Dialog>
            <DialogTrigger asChild>
                <Button>
                    <PlusCircle className="mr-2 h-4 w-4"/>
                    Share a file
                </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>Share a file</DialogTitle>
                    <DialogDescription>
                        Upload a file and share it safely
                    </DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="file" className="">File</Label>
                        <Input id="file" type="file" className="col-span-3"/>
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="username" className="">
                            Password
                        </Label>
                        <Input
                            id="username"
                            defaultValue="0UymxETG$wc)7k8"
                            className="col-span-3"
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="max-downloads" className="">Max downloads</Label>
                        <Select>
                            <SelectTrigger id="max-downloads" className="col-span-3">
                                <SelectValue placeholder="Unlimited" defaultValue="unlimited"/>
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="unlimited">
                                    Unlimited
                                </SelectItem>
                                <SelectItem value="1">1</SelectItem>
                                <SelectItem value="3">3</SelectItem>
                                <SelectItem value="5">5</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="expires-at" className="">Expires at</Label>
                        <Switch id="expires-at" checked={expiresAt} onCheckedChange={setExpiresAt}/>
                    </div>
                    {expiresAt &&
						<div className="grid grid-cols-4 items-center gap-4">
							<Label htmlFor="expires-at-date" className="">Date</Label>
							<DatePickerDemo/>
						</div>
                    }
                </div>
                <DialogFooter>
                    <Button type="submit">Share</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
