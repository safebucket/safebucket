import {FileView} from "@/components/fileview";

export function BucketContent({files}: any) {
    return (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6 gap-4">
            {files.map((file: any) =>
                <FileView key={file.id} file={file}/>
            )}
        </div>
    )
}