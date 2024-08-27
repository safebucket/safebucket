"use client"

import {useParams} from "next/navigation";
import {BucketHeader} from "@/components/bucket/bucket-header";
import {BucketContent} from "@/components/bucket/bucket-content";

const files = [
    {
        id: 0,
        name: "Document.pdf",
        size: "2.3 MB",
        modified: "2023-04-15",
        type: "pdf",
        selected: true,
    },
    {
        id: 1,
        name: "Presentation.pptx",
        size: "5.1 MB",
        modified: "2023-03-28",
        type: "pptx",
        selected: false,
    },
    {
        id: 2,
        name: "Image.jpg",
        size: "1.7 MB",
        modified: "2023-05-02",
        type: "jpg",
        selected: false,
    },
    {
        id: 3,
        name: "Spreadsheet.xlsx",
        size: "3.9 MB",
        modified: "2023-02-10",
        type: "xlsx",
        selected: false,
    },
    {
        id: 4,
        name: "Video.mp4",
        size: "12.4 MB",
        modified: "2023-06-01",
        type: "mp4",
        selected: false,
    },
    {
        id: 5,
        name: "Audio.mp3",
        size: "4.2 MB",
        modified: "2023-01-20",
        type: "mp3",
        selected: false,
    },
    {
        id: 6,
        name: "Document2.pdf",
        size: "1.9 MB",
        modified: "2023-07-05",
        type: "pdf",
        selected: false,
    },
    {
        id: 7,
        name: "Presentation2.pptx",
        size: "6.7 MB",
        modified: "2023-08-15",
        type: "pptx",
        selected: false,
    },
]

export default function Bucket() {
    const {id} = useParams()
    return (
        <div className="">
            <BucketHeader bucketName="Bucket"/>
            <BucketContent files={files}/>
        </div>
    )
}
