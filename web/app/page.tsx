import {FileUp} from "lucide-react";
import Link from "next/link";
import {FileView} from "@/components/fileview";

const activities = [
    {
        id: 1,
        user: "John F.",
        file: "Document.pdf",
        action: "downloaded",
        bucket: "HR bucket",
        modified: "2023-04-15"
    },
    {
        id: 2,
        user: "Pierre R.",
        file: "Presentation.pptx",
        action: "uploaded",
        bucket: "Finance bucket",
        modified: "2023-03-28"
    },
    {
        id: 2,
        user: "Sarah L.",
        file: "Image.jpg",
        action: "uploaded",
        bucket: "Design bucket",
        modified: "2024-08-19"
    },
    {
        id: 2,
        user: "Spreadsheet.xlsx",
        file: "Presentation.pptx",
        action: "uploaded",
        bucket: "Finance bucket",
        modified: "2023-04-15"
    }
]

const files = [
    {
        id: 1,
        name: "Document.pdf",
        size: "2.3 MB",
        modified: "2023-04-15",
        type: "pdf",
        selected: false,
    },
    {
        id: 2,
        name: "Presentation.pptx",
        size: "5.1 MB",
        modified: "2023-03-28",
        type: "pptx",
        selected: false,
    },
    {
        id: 3,
        name: "Image.jpg",
        size: "1.7 MB",
        modified: "2023-05-02",
        type: "jpg",
        selected: false,
    },
    {
        id: 4,
        name: "Spreadsheet.xlsx",
        size: "3.9 MB",
        modified: "2023-02-10",
        type: "xlsx",
        selected: false,
    },
    {
        id: 5,
        name: "Image.jpg",
        size: "1.7 MB",
        modified: "2023-05-02",
        type: "jpg",
        selected: false,
    },
    {
        id: 6,
        name: "Spreadsheet.xlsx",
        size: "3.9 MB",
        modified: "2023-02-10",
        type: "xlsx",
        selected: false,
    },
]

export default function Homepage() {
    return (
        <div className="flex-1 m-6">
            <div className="grid grid-cols-1 gap-8">
                <div className="mb-6">
                    <div className="flex justify-between items-center mb-6">
                        <h1 className="text-2xl font-bold">Recent Files</h1>
                        <Link href="#" className="text-primary hover:underline" prefetch={false}>
                            View all
                        </Link>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                        {files.map((file: any) =>
                            <FileView key={file.id} file={file}/>
                        )}
                    </div>
                </div>
                <div className="flex justify-between items-center">
                    <h1 className="text-2xl font-bold">Latest Activity</h1>
                    <Link href="#" className="text-primary hover:underline" prefetch={false}>
                        View all
                    </Link>
                </div>
                <div className="space-y-4">
                    {activities.map((activity) => (
                        <div
                            key={activity.id}
                            className=" flex items-center gap-4 rounded-md"
                        >
                            <div className="bg-muted rounded-md flex items-center justify-center aspect-square w-12">
                                <FileUp className="w-6 h-6"/>
                            </div>
                            <div className="flex-1">
                                <p className={`font-medium truncate`}>
                                    {activity.user} {activity.action} a file in the {activity.bucket}
                                </p>
                                <p className={`text-sm`}>
                                    {activity.file} - {activity.modified}
                                </p>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    )
}